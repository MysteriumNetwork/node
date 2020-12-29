/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	stdlog "log"
	"path/filepath"
	"strings"
	"time"

	"github.com/mysteriumnetwork/node/core/node"

	"github.com/chzyer/readline"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"

	"github.com/mysteriumnetwork/node/cmd"
	"github.com/mysteriumnetwork/node/cmd/commands/cli/clio"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/datasize"
	"github.com/mysteriumnetwork/node/metadata"
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/services"
	tequilapi_client "github.com/mysteriumnetwork/node/tequilapi/client"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/utils"
)

// CommandName is the name which is used to call this command
const CommandName = "cli"

const serviceHelp = `service <action> [args]
	start	<ProviderID> <ServiceType> [options]
	stop	<ServiceID>
	status	<ServiceID>
	list
	sessions

	example: service start 0x7d5ee3557775aed0b85d691b036769c17349db23 openvpn --openvpn.port=1194 --openvpn.proto=UDP`

// NewCommand constructs CLI based Mysterium UI with possibility to control quiting
func NewCommand() *cli.Command {
	return &cli.Command{
		Name:  CommandName,
		Usage: "Starts a CLI client with a Tequilapi",
		//Before: clicontext.LoadUserConfigQuietly,
		Flags: []cli.Flag{&config.FlagAgreedTermsConditions},
		Action: func(ctx *cli.Context) error {
			config.ParseFlagsNode(ctx)
			nodeOptions := node.GetOptions()
			fmt.Println(nodeOptions)
			client, err := initTequilapiClinet()
			if err != nil {
				return err
			}

			// fix (has to mutate by testnet & stuff)
			dataDir := rConfig.GetStringByFlag(config.FlagDataDir)
			cmdCLI := &cliApp{
				historyFile: filepath.Join(dataDir, ".cli_history"),
				tequilapi:   client,
			}

			cmd.RegisterSignalCallback(utils.SoftKiller(cmdCLI.Kill))

			return describeQuit(cmdCLI.Run(ctx))
		},
	}
}

func initTequilapiClinet() (*tequilapi_client.Client, error) {
	client := tequilapi_client.NewClient(defaultTequilApiAddress, defaultTequilApiPort)

	err := refreshRemoteConfig(client)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func describeQuit(err error) error {
	if err == nil || err == io.EOF || err == readline.ErrInterrupt {
		log.Info().Msg("Stopping application")
		return nil
	}
	log.Error().Err(err).Stack().Msg("Terminating application due to error")
	return err
}

// cliApp describes CLI based Mysterium UI
type cliApp struct {
	historyFile      string
	tequilapi        *tequilapi_client.Client
	fetchedProposals []contract.ProposalDTO
	completer        *readline.PrefixCompleter
	reader           *readline.Instance

	currentConsumerID string
}

const redColor = "\033[31m%s\033[0m"
const identityDefaultPassphrase = ""
const statusConnected = "Connected"

var versionSummary = metadata.VersionAsSummary(metadata.LicenseCopyright(
	"type 'license --warranty'",
	"type 'license --conditions'",
))

func (c *cliApp) handleTOS(ctx *cli.Context) error {
	if ctx.Bool(config.FlagAgreedTermsConditions.Name) {
		c.acceptTOS()
		return nil
	}

	agreedC := rConfig.GetBool(contract.TermsConsumerAgreed)
	if !agreedC {
		return errors.New("You must agree with provider and consumer terms of use in order to use this command")
	}

	agreedP := rConfig.GetBool(contract.TermsProviderAgreed)
	if !agreedP {
		return errors.New("You must agree with provider and consumer terms of use in order to use this command")
	}

	version := rConfig.GetString(contract.TermsVersion)
	if version != metadata.CurrentTermsVersion {
		return fmt.Errorf("You've agreed to terms of use version %s, but version %s is required", version, metadata.CurrentTermsVersion)
	}

	return nil
}

func (c *cliApp) acceptTOS() {
	t := true
	if err := c.tequilapi.UpdateTerms(contract.TermsRequest{
		AgreedConsumer: &t,
		AgreedProvider: &t,
		AgreedVersion:  metadata.CurrentTermsVersion,
	}); err != nil {
		clio.Info("Failed to save terms of use agreement, you will have to re-agree on next launch")
	}
}

// Run runs CLI interface synchronously, in the same thread while blocking it
func (c *cliApp) Run(ctx *cli.Context) (err error) {
	if err := c.handleTOS(ctx); err != nil {
		clio.PrintTOSError(err)
		return nil
	}

	c.completer = newAutocompleter(c.tequilapi, c.fetchedProposals)
	c.fetchedProposals = c.fetchProposals()

	if ctx.Args().Len() > 0 {
		c.handleActions(strings.Join(ctx.Args().Slice(), " "))
		return nil
	}

	c.reader, err = readline.NewEx(&readline.Config{
		Prompt:          fmt.Sprintf(redColor, "» "),
		HistoryFile:     c.historyFile,
		AutoComplete:    c.completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return err
	}
	// TODO Should overtake output of CommandRun
	stdlog.SetOutput(c.reader.Stderr())

	for {
		line, err := c.reader.Readline()
		if err == readline.ErrInterrupt && len(line) > 0 {
			continue
		} else if err != nil {
			c.quit()
			return err
		}

		c.handleActions(line)
	}
}

// Kill stops cli
func (c *cliApp) Kill() error {
	c.reader.Clean()
	return c.reader.Close()
}

func (c *cliApp) handleActions(line string) {
	line = strings.TrimSpace(line)

	staticCmds := []struct {
		command string
		handler func()
	}{
		{"exit", c.quit},
		{"quit", c.quit},
		{"help", c.help},
		{"status", c.status},
		{"healthcheck", c.healthcheck},
		{"nat", c.natStatus},
		{"location", c.location},
		{"disconnect", c.disconnect},
		{"stop", c.stopClient},
	}

	argCmds := []struct {
		command string
		handler func(argsString string)
	}{
		{"connect", c.connect},
		{"identities", c.identities},
		{"order", c.order},
		{"payout", c.payout},
		{"version", c.version},
		{"license", c.license},
		{"proposals", c.proposals},
		{"service", c.service},
		{"stake", c.stake},
		{"mmn", c.mmnApiKey},
	}

	for _, cmd := range staticCmds {
		if line == cmd.command {
			cmd.handler()
			return
		}
	}

	for _, cmd := range argCmds {
		if strings.HasPrefix(line, cmd.command) {
			argsString := strings.TrimSpace(line[len(cmd.command):])
			cmd.handler(argsString)
			return
		}
	}

	if len(line) > 0 {
		c.help()
	}
}

func (c *cliApp) service(argsString string) {
	args := strings.Fields(argsString)
	if len(args) == 0 {
		fmt.Println(serviceHelp)
		return
	}

	action := args[0]
	switch action {
	case "start":
		if len(args) < 3 {
			fmt.Println(serviceHelp)
			return
		}
		c.serviceStart(args[1], args[2], args[3:]...)
	case "stop":
		if len(args) < 2 {
			fmt.Println(serviceHelp)
			return
		}
		c.serviceStop(args[1])
	case "status":
		if len(args) < 2 {
			fmt.Println(serviceHelp)
			return
		}
		c.serviceGet(args[1])
	case "list":
		c.serviceList()
	case "sessions":
		c.serviceSessions()
	default:
		clio.Info(fmt.Sprintf("Unknown action provided: %s", action))
		fmt.Println(serviceHelp)
	}
}

func (c *cliApp) serviceStart(providerID, serviceType string, args ...string) {
	serviceOpts, err := parseStartFlags(serviceType, args...)
	if err != nil {
		clio.Info("Failed to parse service options:", err)
		return
	}

	service, err := c.tequilapi.ServiceStart(contract.ServiceStartRequest{
		ProviderID: providerID,
		Type:       serviceType,
		PaymentMethod: contract.ServicePaymentMethod{
			PriceGB:     serviceOpts.PaymentPricePerGB,
			PriceMinute: serviceOpts.PaymentPricePerMinute,
		},
		AccessPolicies: contract.ServiceAccessPolicies{IDs: serviceOpts.AccessPolicyList},
		Options:        serviceOpts.TypeOptions,
	})
	if err != nil {
		clio.Info("Failed to start service: ", err)
		return
	}

	clio.Status(service.Status,
		"ID: "+service.ID,
		"ProviderID: "+service.Proposal.ProviderID,
		"Type: "+service.Proposal.ServiceType)
}

func (c *cliApp) serviceStop(id string) {
	if err := c.tequilapi.ServiceStop(id); err != nil {
		clio.Info("Failed to stop service: ", err)
		return
	}

	clio.Status("Stopping", "ID: "+id)
}

func (c *cliApp) serviceList() {
	services, err := c.tequilapi.Services()
	if err != nil {
		clio.Info("Failed to get a list of services: ", err)
		return
	}

	for _, service := range services {
		clio.Status(service.Status,
			"ID: "+service.ID,
			"ProviderID: "+service.Proposal.ProviderID,
			"Type: "+service.Proposal.ServiceType)
	}
}

func (c *cliApp) serviceSessions() {
	sessions, err := c.tequilapi.Sessions()
	if err != nil {
		clio.Info("Failed to get a list of sessions: ", err)
		return
	}

	clio.Status("Current sessions", len(sessions.Items))
	for _, session := range sessions.Items {
		clio.Status(
			"ID: "+session.ID,
			"ConsumerID: "+session.ConsumerID,
			fmt.Sprintf("Data: %s/%s", datasize.FromBytes(session.BytesReceived).String(), datasize.FromBytes(session.BytesSent).String()),
			fmt.Sprintf("Tokens: %s", money.New(session.Tokens)),
		)
	}
}

func (c *cliApp) serviceGet(id string) {
	service, err := c.tequilapi.Service(id)
	if err != nil {
		clio.Info("Failed to get service info: ", err)
		return
	}

	clio.Status(service.Status,
		"ID: "+service.ID,
		"ProviderID: "+service.Proposal.ProviderID,
		"Type: "+service.Proposal.ServiceType)
}

func (c *cliApp) connect(argsString string) {
	args := strings.Fields(argsString)

	helpMsg := "Please type in the provider identity. connect <consumer-identity> <provider-identity> <service-type> [dns=auto|provider|system|1.1.1.1] [disable-kill-switch]"
	if len(args) < 3 {
		clio.Info(helpMsg)
		return
	}

	consumerID, providerID, serviceType := args[0], args[1], args[2]

	if !services.IsTypeValid(serviceType) {
		clio.Warn(fmt.Sprintf("Invalid service type, expected one of: %s", strings.Join(services.Types(), ",")))
		return
	}

	var disableKillSwitch bool
	var dns connection.DNSOption
	var err error
	for _, arg := range args[3:] {
		if strings.HasPrefix(arg, "dns=") {
			kv := strings.Split(arg, "=")
			dns, err = connection.NewDNSOption(kv[1])
			if err != nil {
				clio.Warn("Invalid value: ", err)
				clio.Info(helpMsg)
				return
			}
			continue
		}
		switch arg {
		case "disable-kill-switch":
			disableKillSwitch = true
		default:
			clio.Warn("Unexpected arg:", arg)
			clio.Info(helpMsg)
			return
		}
	}

	connectOptions := contract.ConnectOptions{
		DNS:               dns,
		DisableKillSwitch: disableKillSwitch,
	}

	clio.Status("CONNECTING", "from:", consumerID, "to:", providerID)

	hermesID := rConfig.GetStringByFlag(config.FlagHermesID)

	// Dont throw an error here incase user identity has a password on it
	// or we failed to randomly unlock it. We can still try to connect
	// if identity it locked, it will notify us anyway.
	_ = c.tequilapi.Unlock(consumerID, "")

	_, err = c.tequilapi.ConnectionCreate(consumerID, providerID, hermesID, serviceType, connectOptions)
	if err != nil {
		clio.Error(err)
		return
	}

	c.currentConsumerID = consumerID

	clio.Success("Connected.")
}

func (c *cliApp) payout(argsString string) {
	args := strings.Fields(argsString)

	const usage = "payout command:\n    set"
	if len(args) == 0 {
		clio.Info(usage)
		return
	}

	action := args[0]
	switch action {
	case "set":
		payoutSignature := "payout set <identity> <ethAddress>"
		if len(args) < 2 {
			clio.Info("Please provide identity. You can select one by pressing tab.\n", payoutSignature)
			return
		}

		var identity, ethAddress string
		if len(args) > 2 {
			identity, ethAddress = args[1], args[2]
		} else {
			clio.Info("Please type in identity and Ethereum address.\n", payoutSignature)
			return
		}

		err := c.tequilapi.Payout(identity, ethAddress)
		if err != nil {
			clio.Warn(err)
			return
		}

		clio.Success(fmt.Sprintf("Payout address %s registered.", ethAddress))
	default:
		clio.Warnf("Unknown sub-command '%s'\n", action)
		fmt.Println(usage)
		return
	}
}

func (c *cliApp) mmnApiKey(argsString string) {
	args := strings.Fields(argsString)

	var profileUrl = rConfig.GetStringByFlag(config.FlagMMNAddress) + "user/profile"
	var usage = "Set MMN's API key and claim this node:\nmmn <api-key>\nTo get the token, visit: " + profileUrl + "\n"

	if len(args) == 0 {
		clio.Info(usage)
		return
	}

	apiKey := args[0]

	err := c.tequilapi.SetMMNApiKey(contract.MMNApiKeyRequest{
		ApiKey: apiKey,
	})

	if err != nil {
		clio.Warn(err)
		return
	}

	clio.Success(fmt.Sprint("MMN API key configured."))
}

func (c *cliApp) disconnect() {
	err := c.tequilapi.ConnectionDestroy()
	if err != nil {
		clio.Warn(err)
		return
	}
	c.currentConsumerID = ""
	clio.Success("Disconnected.")
}

func (c *cliApp) status() {
	status, err := c.tequilapi.ConnectionStatus()
	if err != nil {
		clio.Warn(err)
	} else {
		clio.Info("Status:", status.Status)
		clio.Info("SID:", status.SessionID)
	}

	ip, err := c.tequilapi.ConnectionIP()
	if err != nil {
		clio.Warn(err)
	} else {
		clio.Info("IP:", ip.IP)
	}

	location, err := c.tequilapi.ConnectionLocation()
	if err != nil {
		clio.Warn(err)
	} else {
		clio.Info(fmt.Sprintf("Location: %s, %s (%s - %s)", location.City, location.Country, location.UserType, location.ISP))
	}

	if status.Status == statusConnected {
		clio.Info("Proposal:", status.Proposal)

		statistics, err := c.tequilapi.ConnectionStatistics()
		if err != nil {
			clio.Warn(err)
		} else {
			clio.Info(fmt.Sprintf("Connection duration: %s", time.Duration(statistics.Duration)*time.Second))
			clio.Info(fmt.Sprintf("Data: %s/%s", datasize.FromBytes(statistics.BytesReceived), datasize.FromBytes(statistics.BytesSent)))
			clio.Info(fmt.Sprintf("Throughput: %s/%s", datasize.BitSpeed(statistics.ThroughputReceived), datasize.BitSpeed(statistics.ThroughputSent)))
			clio.Info(fmt.Sprintf("Spent: %s", money.New(statistics.TokensSpent)))
		}
	}
}

func (c *cliApp) healthcheck() {
	healthcheck, err := c.tequilapi.Healthcheck()
	if err != nil {
		clio.Warn(err)
		return
	}

	clio.Info(fmt.Sprintf("Uptime: %v", healthcheck.Uptime))
	clio.Info(fmt.Sprintf("Process: %v", healthcheck.Process))
	clio.Info(fmt.Sprintf("Version: %v", healthcheck.Version))
	buildString := metadata.FormatString(healthcheck.BuildInfo.Commit, healthcheck.BuildInfo.Branch, healthcheck.BuildInfo.BuildNumber)
	clio.Info(buildString)
}

func (c *cliApp) natStatus() {
	status, err := c.tequilapi.NATStatus()
	if err != nil {
		clio.Warn("Failed to retrieve NAT traversal status:", err)
		return
	}

	if status.Error == "" {
		clio.Infof("NAT traversal status: %q\n", status.Status)
	} else {
		clio.Infof("NAT traversal status: %q (error: %q)\n", status.Status, status.Error)
	}
}

func (c *cliApp) proposals(filter string) {
	proposals := c.fetchProposals()
	c.fetchedProposals = proposals

	filterMsg := ""
	if filter != "" {
		filterMsg = fmt.Sprintf("(filter: '%s')", filter)
	}
	clio.Info(fmt.Sprintf("Found %v proposals %s", len(proposals), filterMsg))

	for _, proposal := range proposals {
		country := proposal.ServiceDefinition.LocationOriginate.Country
		if country == "" {
			country = "Unknown"
		}

		var policies []string
		if proposal.AccessPolicies != nil {
			for _, policy := range *proposal.AccessPolicies {
				policies = append(policies, policy.ID)
			}
		}

		msg := fmt.Sprintf("- provider id: %v\ttype: %v\tcountry: %v\taccess policies: %v", proposal.ProviderID, proposal.ServiceType, country, strings.Join(policies, ","))

		if filter == "" ||
			strings.Contains(proposal.ProviderID, filter) ||
			strings.Contains(country, filter) {

			clio.Info(msg)
		}
	}
}

func (c *cliApp) fetchProposals() []contract.ProposalDTO {
	upperTimeBound := rConfig.GetBigIntByFlag(config.FlagPaymentsConsumerPricePerMinuteUpperBound)
	lowerTimeBound := rConfig.GetBigIntByFlag(config.FlagPaymentsConsumerPricePerMinuteLowerBound)
	upperGBBound := rConfig.GetBigIntByFlag(config.FlagPaymentsConsumerPricePerGBUpperBound)
	lowerGBBound := rConfig.GetBigIntByFlag(config.FlagPaymentsConsumerPricePerGBLowerBound)
	proposals, err := c.tequilapi.ProposalsByPrice(lowerTimeBound, upperTimeBound, lowerGBBound, upperGBBound)
	if err != nil {
		clio.Warn(err)
		return []contract.ProposalDTO{}
	}
	return proposals
}

func (c *cliApp) location() {
	location, err := c.tequilapi.OriginLocation()
	if err != nil {
		clio.Warn(err)
		return
	}

	clio.Info(fmt.Sprintf("Location: %s, %s (%s - %s)", location.City, location.Country, location.UserType, location.ISP))
}

func (c *cliApp) help() {
	clio.Info("Mysterium CLI commands:")
	fmt.Println(c.completer.Tree("  "))
}

// quit stops cli and client commands and exits application
func (c *cliApp) quit() {
	stop := utils.SoftKiller(c.Kill)
	stop()
}

func (c *cliApp) stopClient() {
	err := c.tequilapi.Stop()
	if err != nil {
		clio.Warn("Cannot stop client:", err)
	}
	clio.Success("Client stopped")
}

func (c *cliApp) version(argsString string) {
	fmt.Println(versionSummary)
}

func (c *cliApp) license(argsString string) {
	if argsString == "warranty" {
		fmt.Print(metadata.LicenseWarranty)
	} else if argsString == "conditions" {
		fmt.Print(metadata.LicenseConditions)
	} else {
		clio.Info("identities command:\n    warranty\n    conditions")
	}
}

func getIdentityOptionList(tequilapi *tequilapi_client.Client) func(string) []string {
	return func(line string) []string {
		var identities []string
		ids, err := tequilapi.GetIdentities()
		if err != nil {
			clio.Warn(err)
			return identities
		}
		for _, id := range ids {
			identities = append(identities, id.Address)
		}

		return identities
	}
}

func getProposalOptionList(proposals []contract.ProposalDTO) func(string) []string {
	return func(line string) []string {
		var providerIDS []string
		for _, proposal := range proposals {
			providerIDS = append(providerIDS, proposal.ProviderID)
		}
		return providerIDS
	}
}

func newAutocompleter(tequilapi *tequilapi_client.Client, proposals []contract.ProposalDTO) *readline.PrefixCompleter {
	connectOpts := []readline.PrefixCompleterInterface{
		readline.PcItem("dns=auto"),
		readline.PcItem("dns=provider"),
		readline.PcItem("dns=system"),
		readline.PcItem("dns=1.1.1.1"),
	}
	return readline.NewPrefixCompleter(
		readline.PcItem(
			"connect",
			readline.PcItemDynamic(
				getIdentityOptionList(tequilapi),
				readline.PcItemDynamic(
					getProposalOptionList(proposals),
					readline.PcItem("noop", connectOpts...),
					readline.PcItem("openvpn", connectOpts...),
					readline.PcItem("wireguard", connectOpts...),
				),
			),
		),
		readline.PcItem(
			"service",
			readline.PcItem("start", readline.PcItemDynamic(
				getIdentityOptionList(tequilapi),
				readline.PcItem("noop"),
				readline.PcItem("openvpn"),
				readline.PcItem("wireguard"),
			)),
			readline.PcItem("stop"),
			readline.PcItem("list"),
			readline.PcItem("status"),
			readline.PcItem("sessions"),
		),
		readline.PcItem(
			"identities",
			readline.PcItem("list"),
			readline.PcItem("get", readline.PcItemDynamic(getIdentityOptionList(tequilapi))),
			readline.PcItem("new"),
			readline.PcItem("unlock", readline.PcItemDynamic(getIdentityOptionList(tequilapi))),
			readline.PcItem("register", readline.PcItemDynamic(getIdentityOptionList(tequilapi))),
			readline.PcItem("beneficiary", readline.PcItemDynamic(getIdentityOptionList(tequilapi))),
			readline.PcItem("settle", readline.PcItemDynamic(getIdentityOptionList(tequilapi))),
			readline.PcItem("referralcode", readline.PcItemDynamic(getIdentityOptionList(tequilapi))),
		),
		readline.PcItem("status"),
		readline.PcItem(
			"stake",
			readline.PcItem("increase"),
			readline.PcItem("decrease"),
		),
		readline.PcItem("orders",
			readline.PcItem("create"),
			readline.PcItem("get"),
			readline.PcItem("get-all"),
			readline.PcItem("currencies"),
		),
		readline.PcItem("healthcheck"),
		readline.PcItem("nat"),
		readline.PcItem("proposals"),
		readline.PcItem("location"),
		readline.PcItem("disconnect"),
		readline.PcItem("mmn"),
		readline.PcItem("help"),
		readline.PcItem("quit"),
		readline.PcItem("stop"),
		readline.PcItem(
			"payout",
			readline.PcItem("set", readline.PcItemDynamic(getIdentityOptionList(tequilapi))),
		),
		readline.PcItem(
			"license",
			readline.PcItem("warranty"),
			readline.PcItem("conditions"),
		),
	)
}

func parseStartFlags(serviceType string, args ...string) (services.StartOptions, error) {
	var flags []cli.Flag
	config.RegisterFlagsServiceStart(&flags)
	config.RegisterFlagsServiceOpenvpn(&flags)
	config.RegisterFlagsServiceWireguard(&flags)
	config.RegisterFlagsServiceNoop(&flags)

	set := flag.NewFlagSet("", flag.ContinueOnError)
	for _, f := range flags {
		f.Apply(set)
	}
	if err := set.Parse(args); err != nil {
		return services.StartOptions{}, err
	}

	ctx := cli.NewContext(nil, set, nil)
	config.ParseFlagsServiceStart(ctx)
	config.ParseFlagsServiceOpenvpn(ctx)
	config.ParseFlagsServiceWireguard(ctx)
	config.ParseFlagsServiceNoop(ctx)

	return services.GetStartOptions(serviceType)
}
