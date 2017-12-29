package command_run

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/mysterium/node/client_connection"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats_dialog"
	"github.com/mysterium/node/communication/nats_discovery"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/tequilapi"
	"github.com/mysterium/node/tequilapi/endpoints"
)

type CommandRun struct {
	//TODO this must disappear or become a private field
	MysteriumClient server.Client

	connectionManager client_connection.Manager

	httpApiServer tequilapi.ApiServer
}

func NewCommand(options CommandOptions) *CommandRun {
	nats_discovery.Bootstrap()
	openvpn.Bootstrap()

	mysteriumClient := server.NewClient()

	keystoreInstance := keystore.NewKeyStore(options.DirectoryKeystore, keystore.StandardScryptN, keystore.StandardScryptP)

	identityManager := identity.NewIdentityManager(keystoreInstance)

	dialogEstablisherFactory := func(identity identity.Identity) communication.DialogEstablisher {
		return nats_dialog.NewDialogEstablisher(identity)
	}

	vpnClientFactory := client_connection.ConfigureVpnClientFactory(mysteriumClient, options.DirectoryRuntime)

	connectionManager := client_connection.NewManager(mysteriumClient, dialogEstablisherFactory, vpnClientFactory)

	router := tequilapi.NewApiRouter()
	endpoints.RegisterIdentitiesEndpoint(router, identityManager, mysteriumClient)
	endpoints.RegisterConnectionEndpoint(router, connectionManager)

	httpApiServer := tequilapi.NewServer(options.TequilaApiAddress, options.TequilaApiPort, router)

	return &CommandRun{
		mysteriumClient,
		connectionManager,
		httpApiServer,
	}
}

func (cmd *CommandRun) Run() error {
	err := cmd.httpApiServer.StartServing()
	if err != nil {
		return err
	}
	port, err := cmd.httpApiServer.Port()
	if err != nil {
		return err
	}

	fmt.Printf("Api started on: %d\n", port)
	return nil
}

func (cmd *CommandRun) Wait() error {
	return cmd.httpApiServer.Wait()
}

func (cmd *CommandRun) Kill() {
	cmd.httpApiServer.Stop()
	cmd.connectionManager.Disconnect()
}
