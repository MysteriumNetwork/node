/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package mysterium

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"

	"github.com/mysteriumnetwork/node/cmd"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/consumer/entertainment"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/core/quality"
	"github.com/mysteriumnetwork/node/core/state"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/feedback"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/identity/selector"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/metadata"
	"github.com/mysteriumnetwork/node/pilvytis"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/mysteriumnetwork/node/router"
	"github.com/mysteriumnetwork/node/services/wireguard"
	wireguard_connection "github.com/mysteriumnetwork/node/services/wireguard/connection"
	"github.com/mysteriumnetwork/node/session/pingpong"
	"github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/mysteriumnetwork/payments/crypto"
)

// MobileNode represents node object tuned for mobile device.
type MobileNode struct {
	shutdown                  func() error
	node                      *cmd.Node
	stateKeeper               *state.Keeper
	connectionManager         connection.Manager
	locationResolver          *location.Cache
	identitySelector          selector.Handler
	signerFactory             identity.SignerFactory
	identityMover             *identity.Mover
	ipResolver                ip.Resolver
	eventBus                  eventbus.EventBus
	connectionRegistry        *connection.Registry
	proposalsManager          *proposalsManager
	feedbackReporter          *feedback.Reporter
	transactor                *registry.Transactor
	identityRegistry          registry.IdentityRegistry
	identityChannelCalculator *pingpong.AddressProvider
	consumerBalanceTracker    *pingpong.ConsumerBalanceTracker
	pilvytis                  *pilvytis.Service
	chainID                   int64
	startTime                 time.Time
	sessionStorage            SessionStorage
	entertainmentEstimator    *entertainment.Estimator
	residentCountry           *identity.ResidentCountry
	filterPresetStorage       *proposal.FilterPresetStorage
}

// MobileNodeOptions contains common mobile node options.
type MobileNodeOptions struct {
	Testnet3                       bool
	Localnet                       bool
	ExperimentNATPunching          bool
	KeepConnectedOnFail            bool
	MysteriumAPIAddress            string
	BrokerAddresses                []string
	EtherClientRPCL1               []string
	EtherClientRPCL2               []string
	FeedbackURL                    string
	QualityOracleURL               string
	IPDetectorURL                  string
	LocationDetectorURL            string
	TransactorEndpointAddress      string
	HermesEndpointAddress          string
	Chain1ID                       int64
	Chain2ID                       int64
	ActiveChainID                  int64
	PilvytisAddress                string
	MystSCAddress                  string
	RegistrySCAddress              string
	HermesSCAddress                string
	ChannelImplementationSCAddress string
	CacheTTLSeconds                int
}

// ConsumerPaymentConfig defines consumer side payment configuration
type ConsumerPaymentConfig struct {
	PriceGiBMax  string
	PriceHourMax string
}

// DefaultNodeOptions returns default options.
func DefaultNodeOptions() *MobileNodeOptions {
	return &MobileNodeOptions{
		Testnet3:                       true,
		ExperimentNATPunching:          true,
		KeepConnectedOnFail:            true,
		MysteriumAPIAddress:            metadata.Testnet3Definition.MysteriumAPIAddress,
		BrokerAddresses:                metadata.Testnet3Definition.BrokerAddresses,
		EtherClientRPCL1:               metadata.Testnet3Definition.Chain1.EtherClientRPC,
		EtherClientRPCL2:               metadata.Testnet3Definition.Chain2.EtherClientRPC,
		FeedbackURL:                    "https://feedback.mysterium.network",
		QualityOracleURL:               "https://testnet3-quality.mysterium.network/api/v2",
		IPDetectorURL:                  "https://testnet3-location.mysterium.network/api/v1/location",
		LocationDetectorURL:            "https://testnet3-location.mysterium.network/api/v1/location",
		TransactorEndpointAddress:      metadata.Testnet3Definition.TransactorAddress,
		ActiveChainID:                  metadata.Testnet3Definition.DefaultChainID,
		Chain1ID:                       metadata.Testnet3Definition.Chain1.ChainID,
		Chain2ID:                       metadata.Testnet3Definition.Chain2.ChainID,
		PilvytisAddress:                metadata.Testnet3Definition.PilvytisAddress,
		MystSCAddress:                  metadata.Testnet3Definition.Chain1.MystAddress,
		RegistrySCAddress:              metadata.Testnet3Definition.Chain1.RegistryAddress,
		HermesSCAddress:                metadata.Testnet3Definition.Chain1.HermesID,
		ChannelImplementationSCAddress: metadata.Testnet3Definition.Chain1.ChannelImplAddress,
		CacheTTLSeconds:                5,
	}
}

// NewNode function creates new Node.
func NewNode(appPath string, options *MobileNodeOptions) (*MobileNode, error) {
	var di cmd.Dependencies

	if appPath == "" {
		return nil, errors.New("node app path is required")
	}

	dataDir := filepath.Join(appPath, ".mysterium")
	currentDir := appPath
	if err := loadUserConfig(dataDir); err != nil {
		return nil, err
	}

	config.Current.SetDefault(config.FlagChainID.Name, options.ActiveChainID)
	config.Current.SetDefault(config.FlagKeepConnectedOnFail.Name, options.KeepConnectedOnFail)
	config.Current.SetDefault(config.FlagDefaultCurrency.Name, metadata.DefaultNetwork.DefaultCurrency)
	config.Current.SetDefault(config.FlagSTUNservers.Name, []string{"stun.l.google.com:19302", "stun1.l.google.com:19302", "stun2.l.google.com:19302"})

	network := node.OptionsNetwork{
		Testnet3:              options.Testnet3,
		Localnet:              options.Localnet,
		ExperimentNATPunching: options.ExperimentNATPunching,
		MysteriumAPIAddress:   options.MysteriumAPIAddress,
		BrokerAddresses:       options.BrokerAddresses,
		EtherClientRPCL1:      options.EtherClientRPCL1,
		EtherClientRPCL2:      options.EtherClientRPCL2,
		ChainID:               options.ActiveChainID,
		DNSMap: map[string][]string{
			"testnet3-location.mysterium.network": {"167.233.11.60"},
			"testnet3-quality.mysterium.network":  {"167.233.11.60"},
			"feedback.mysterium.network":          {"116.203.17.150"},
			"api.ipify.org": {
				"54.204.14.42", "54.225.153.147", "54.235.83.248", "54.243.161.145",
				"23.21.109.69", "23.21.126.66",
				"50.19.252.36",
				"174.129.214.20",
			},
		},
	}
	logOptions := logconfig.LogOptions{
		LogLevel: zerolog.DebugLevel,
		LogHTTP:  false,
		Filepath: filepath.Join(dataDir, "mysterium-node"),
	}

	nodeOptions := node.Options{
		LogOptions: logOptions,
		Directories: node.OptionsDirectory{
			Data:     dataDir,
			Storage:  filepath.Join(dataDir, "db"),
			Keystore: filepath.Join(dataDir, "keystore"),
			Runtime:  currentDir,
		},

		TequilapiEnabled:        false,
		SwarmDialerDNSHeadstart: time.Millisecond * 1500,
		Keystore: node.OptionsKeystore{
			UseLightweight: true,
		},
		UI: node.OptionsUI{
			UIEnabled: false,
		},
		FeedbackURL:    options.FeedbackURL,
		OptionsNetwork: network,
		Quality: node.OptionsQuality{
			Type:    node.QualityTypeMORQA,
			Address: options.QualityOracleURL,
		},
		Discovery: node.OptionsDiscovery{
			Types:        []node.DiscoveryType{node.DiscoveryTypeAPI},
			Address:      network.MysteriumAPIAddress,
			FetchEnabled: false,
			DHT: node.OptionsDHT{
				Address:        "0.0.0.0",
				Port:           0,
				Protocol:       "tcp",
				BootstrapPeers: []string{},
			},
		},
		Location: node.OptionsLocation{
			IPDetectorURL: options.IPDetectorURL,
			Type:          node.LocationTypeOracle,
			Address:       options.LocationDetectorURL,
		},
		Transactor: node.OptionsTransactor{
			TransactorEndpointAddress:       options.TransactorEndpointAddress,
			ProviderMaxRegistrationAttempts: 10,
			ProviderRegistrationRetryDelay:  time.Minute * 3,
		},
		Payments: node.OptionsPayments{
			MaxAllowedPaymentPercentile:    1500,
			BCTimeout:                      time.Second * 30,
			HermesPromiseSettlingThreshold: 0.1,
			SettlementTimeout:              time.Hour * 2,
		},
		Chains: node.OptionsChains{
			Chain1: metadata.ChainDefinition{
				RegistryAddress:    options.RegistrySCAddress,
				HermesID:           options.HermesSCAddress,
				ChannelImplAddress: options.ChannelImplementationSCAddress,
				MystAddress:        options.MystSCAddress,
				ChainID:            options.Chain1ID,
			},
			Chain2: metadata.ChainDefinition{
				RegistryAddress:    options.RegistrySCAddress,
				HermesID:           options.HermesSCAddress,
				ChannelImplAddress: options.ChannelImplementationSCAddress,
				MystAddress:        options.MystSCAddress,
				ChainID:            options.Chain2ID,
			},
		},
		Consumer:        true,
		P2PPorts:        port.UnspecifiedRange(),
		PilvytisAddress: options.PilvytisAddress,
	}

	err := di.Bootstrap(nodeOptions)
	if err != nil {
		return nil, fmt.Errorf("could not bootstrap dependencies: %w", err)
	}

	mobileNode := &MobileNode{
		shutdown:                  di.Shutdown,
		node:                      di.Node,
		stateKeeper:               di.StateKeeper,
		connectionManager:         di.ConnectionManager,
		locationResolver:          di.LocationResolver,
		identitySelector:          di.IdentitySelector,
		signerFactory:             di.SignerFactory,
		ipResolver:                di.IPResolver,
		eventBus:                  di.EventBus,
		connectionRegistry:        di.ConnectionRegistry,
		feedbackReporter:          di.Reporter,
		transactor:                di.Transactor,
		identityRegistry:          di.IdentityRegistry,
		consumerBalanceTracker:    di.ConsumerBalanceTracker,
		identityChannelCalculator: di.AddressProvider,
		proposalsManager: newProposalsManager(
			di.ProposalRepository,
			di.FilterPresetStorage,
			time.Duration(options.CacheTTLSeconds)*time.Second,
		),
		pilvytis:       di.Pilvytis,
		startTime:      time.Now(),
		chainID:        nodeOptions.OptionsNetwork.ChainID,
		sessionStorage: di.SessionStorage,
		identityMover:  di.IdentityMover,
		entertainmentEstimator: entertainment.NewEstimator(
			config.FlagPaymentPriceGiB.Value,
			config.FlagPaymentPriceHour.Value,
		),
		residentCountry:     di.ResidentCountry,
		filterPresetStorage: di.FilterPresetStorage,
	}

	return mobileNode, nil
}

// GetDefaultCurrency returns the current default currency set.
func (mb *MobileNode) GetDefaultCurrency() string {
	return config.Current.GetString(config.FlagDefaultCurrency.Name)
}

// ProposalChangeCallback represents proposal callback.
type ProposalChangeCallback interface {
	OnChange(proposal []byte)
}

// GetLocationResponse represents location response.
type GetLocationResponse struct {
	IP      string
	Country string
}

// GetLocation return current location including country and IP.
func (mb *MobileNode) GetLocation() (*GetLocationResponse, error) {
	c := requests.NewHTTPClientWithTransport(http.DefaultTransport.(*http.Transport), 30*time.Second)
	resolver := location.NewOracleResolver(c, DefaultNodeOptions().LocationDetectorURL)
	loc, err := resolver.DetectLocation()
	// TODO this is temporary workaround to show correct location on Android.
	// This needs to be fixed on the di level to make sure we are using correct resolver in transport and in visual part.
	// loc, err := mb.locationResolver.DetectLocation()
	if err != nil {
		return nil, fmt.Errorf("could not get location: %w", err)
	}

	return &GetLocationResponse{
		IP:      loc.IP,
		Country: loc.Country,
	}, nil
}

// GetStatusResponse represents status response.
type GetStatusResponse struct {
	State       string
	ProviderID  string
	ServiceType string
}

// GetStatus returns current connection state and provider info if connected to VPN.
func (mb *MobileNode) GetStatus() *GetStatusResponse {
	status := mb.connectionManager.Status()

	return &GetStatusResponse{
		State:       string(status.State),
		ProviderID:  status.Proposal.ProviderID,
		ServiceType: status.Proposal.ServiceType,
	}
}

// StatisticsChangeCallback represents statistics callback.
type StatisticsChangeCallback interface {
	OnChange(duration int64, bytesReceived int64, bytesSent int64, tokensSpent float64)
}

// RegisterStatisticsChangeCallback registers callback which is called on active connection
// statistics change.
func (mb *MobileNode) RegisterStatisticsChangeCallback(cb StatisticsChangeCallback) {
	_ = mb.eventBus.SubscribeAsync(connectionstate.AppTopicConnectionStatistics, func(e connectionstate.AppEventConnectionStatistics) {
		var tokensSpent float64
		if mb.stateKeeper.GetState().Connection.Invoice.AgreementTotal != nil {
			tokensSpent = crypto.BigMystToFloat(mb.stateKeeper.GetState().Connection.Invoice.AgreementTotal)
		}

		cb.OnChange(int64(e.SessionInfo.Duration().Seconds()), int64(e.Stats.BytesReceived), int64(e.Stats.BytesSent), tokensSpent)
	})
}

// ConnectionStatusChangeCallback represents status callback.
type ConnectionStatusChangeCallback interface {
	OnChange(status string)
}

// RegisterConnectionStatusChangeCallback registers callback which is called on active connection
// status change.
func (mb *MobileNode) RegisterConnectionStatusChangeCallback(cb ConnectionStatusChangeCallback) {
	_ = mb.eventBus.SubscribeAsync(connectionstate.AppTopicConnectionState, func(e connectionstate.AppEventConnectionState) {
		cb.OnChange(string(e.State))
	})
}

// BalanceChangeCallback represents balance change callback.
type BalanceChangeCallback interface {
	OnChange(identityAddress string, balance float64)
}

// RegisterBalanceChangeCallback registers callback which is called on identity balance change.
func (mb *MobileNode) RegisterBalanceChangeCallback(cb BalanceChangeCallback) {
	_ = mb.eventBus.SubscribeAsync(event.AppTopicBalanceChanged, func(e event.AppEventBalanceChanged) {
		balance := crypto.BigMystToFloat(e.Current)
		cb.OnChange(e.Identity.Address, balance)
	})
}

// ConnectRequest represents connect request.
/*
 * DNSOption:
 *	- "auto" (default) tries the following with fallbacks: provider's DNS -> client's system DNS -> public DNS
 *  - "provider" uses DNS servers from provider's system configuration
 *  - "system" uses DNS servers from client's system configuration
 */
type ConnectRequest struct {
	IdentityAddress   string
	ProviderID        string
	ServiceType       string
	DNSOption         string
	DisableKillSwitch bool
	ForceReconnect    bool
}

func (cr *ConnectRequest) dnsOption() (connection.DNSOption, error) {
	if len(cr.DNSOption) > 0 {
		return connection.NewDNSOption(cr.DNSOption)
	}

	return connection.DNSOptionAuto, nil
}

// ConnectResponse represents connect response with optional error code and message.
type ConnectResponse struct {
	ErrorCode    string
	ErrorMessage string
}

const (
	connectErrInvalidProposal     = "InvalidProposal"
	connectErrInsufficientBalance = "InsufficientBalance"
	connectErrUnknown             = "Unknown"
)

// Connect connects to given provider.
func (mb *MobileNode) Connect(req *ConnectRequest) *ConnectResponse {
	proposal, err := mb.proposalsManager.repository.Proposal(market.ProposalID{
		ProviderID:  req.ProviderID,
		ServiceType: req.ServiceType,
	})

	qualityEvent := quality.ConnectionEvent{
		ServiceType: req.ServiceType,
		ConsumerID:  req.IdentityAddress,
		ProviderID:  req.ProviderID,
	}

	if err != nil {
		qualityEvent.Stage = quality.StageGetProposal
		qualityEvent.Error = err.Error()
		mb.eventBus.Publish(quality.AppTopicConnectionEvents, qualityEvent)

		return &ConnectResponse{
			ErrorCode:    connectErrInvalidProposal,
			ErrorMessage: err.Error(),
		}
	}

	dnsOption, err := req.dnsOption()
	if err != nil {
		return &ConnectResponse{
			ErrorCode:    connectErrUnknown,
			ErrorMessage: err.Error(),
		}
	}
	connectOptions := connection.ConnectParams{
		DisableKillSwitch: req.DisableKillSwitch,
		DNS:               dnsOption,
	}

	hermes, err := mb.identityChannelCalculator.GetActiveHermes(mb.chainID)
	if err != nil {
		return &ConnectResponse{
			ErrorCode:    connectErrUnknown,
			ErrorMessage: err.Error(),
		}
	}

	if err := mb.connectionManager.Connect(identity.FromAddress(req.IdentityAddress), hermes, *proposal, connectOptions); err != nil {
		qualityEvent.Stage = quality.StageConnectionUnknownError
		qualityEvent.Error = err.Error()
		mb.eventBus.Publish(quality.AppTopicConnectionEvents, qualityEvent)

		if errors.Is(err, connection.ErrInsufficientBalance) {
			return &ConnectResponse{
				ErrorCode: connectErrInsufficientBalance,
			}
		}

		return &ConnectResponse{
			ErrorCode:    connectErrUnknown,
			ErrorMessage: err.Error(),
		}
	}

	qualityEvent.Stage = quality.StageConnectionOK
	mb.eventBus.Publish(quality.AppTopicConnectionEvents, qualityEvent)

	return &ConnectResponse{}
}

// Reconnect is deprecated, we are doing reconnect now in the connection manager.
// TODO remove this from mobile app and here too.
func (mb *MobileNode) Reconnect(req *ConnectRequest) *ConnectResponse {
	return &ConnectResponse{}
}

// Disconnect disconnects or cancels current connection.
func (mb *MobileNode) Disconnect() error {
	if err := mb.connectionManager.Disconnect(); err != nil {
		return fmt.Errorf("could not disconnect: %w", err)
	}

	return nil
}

// GetBalanceRequest represents balance request.
type GetBalanceRequest struct {
	IdentityAddress string
}

// GetBalanceResponse represents balance response.
type GetBalanceResponse struct {
	Balance float64
}

// GetBalance returns current balance.
func (mb *MobileNode) GetBalance(req *GetBalanceRequest) (*GetBalanceResponse, error) {
	balance := mb.consumerBalanceTracker.GetBalance(mb.chainID, identity.FromAddress(req.IdentityAddress))
	b := crypto.BigMystToFloat(balance)

	return &GetBalanceResponse{Balance: b}, nil
}

// SendFeedbackRequest represents user feedback request.
type SendFeedbackRequest struct {
	Email       string
	Description string
}

// SendFeedback sends user feedback via feedback reported.
func (mb *MobileNode) SendFeedback(req *SendFeedbackRequest) error {
	report := feedback.UserReport{
		Email:       req.Email,
		Description: req.Description,
	}

	result, err := mb.feedbackReporter.NewIssue(report)
	if err != nil {
		return fmt.Errorf("could not create user report: %w", err)
	}

	if !result.Success {
		return errors.New("user report sent but got error response")
	}

	return nil
}

// Shutdown function stops running mobile node.
func (mb *MobileNode) Shutdown() error {
	return mb.shutdown()
}

// WaitUntilDies function returns when node stops.
func (mb *MobileNode) WaitUntilDies() error {
	return mb.node.Wait()
}

// OverrideWireguardConnection overrides default wireguard connection implementation to more mobile adapted one.
func (mb *MobileNode) OverrideWireguardConnection(wgTunnelSetup WireguardTunnelSetup) {
	wireguard.Bootstrap()

	factory := func() (connection.Connection, error) {
		opts := wireGuardOptions{
			statsUpdateInterval: 1 * time.Second,
			handshakeTimeout:    1 * time.Minute,
		}

		return NewWireGuardConnection(
			opts,
			newWireguardDevice(wgTunnelSetup),
			mb.ipResolver,
			wireguard_connection.NewHandshakeWaiter(),
		)
	}
	mb.connectionRegistry.Register(wireguard.ServiceType, factory)

	router.SetProtectFunc(wgTunnelSetup.Protect)
}

// HealthCheckData represents node health check info.
type HealthCheckData struct {
	Uptime    string     `json:"uptime"`
	Version   string     `json:"version"`
	BuildInfo *BuildInfo `json:"build_info"`
}

// BuildInfo represents node build info.
type BuildInfo struct {
	Commit      string `json:"commit"`
	Branch      string `json:"branch"`
	BuildNumber string `json:"build_number"`
}

// HealthCheck returns node health check data.
func (mb *MobileNode) HealthCheck() *HealthCheckData {
	return &HealthCheckData{
		Uptime:  time.Since(mb.startTime).String(),
		Version: metadata.VersionAsString(),
		BuildInfo: &BuildInfo{
			Commit:      metadata.BuildCommit,
			Branch:      metadata.BuildBranch,
			BuildNumber: metadata.BuildNumber,
		},
	}
}
