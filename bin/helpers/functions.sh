#!/bin/bash

# Map environment variables to flags for Golang linker's -ldflags usage
function get_linker_ldflags {
    echo "
        -X 'github.com/mysterium/node/server.mysteriumApiUrl=${MYSTERIUM_API_URL}'
        -X 'github.com/mysterium/node/communication/nats/discovery.natsServerIp=${NATS_SERVER_IP}'
    "
}