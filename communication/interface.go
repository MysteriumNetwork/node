package communication

import dto_discovery "github.com/mysterium/node/service_discovery/dto"

// DialogWaiter defines server which:
//   - waits and servers incoming dialog requests
//   - negotiates with Dialog initiator
//   - finally creates Dialog, when it is accepted
type DialogWaiter interface {
	ServeDialogs(sessionCreateConsumer RequestConsumer) error
	Stop() error
}

// DialogEstablisher interface defines client which:
//   - initiates Dialog requests to network
//   - creates Dialog, when it is negotiated
type DialogEstablisher interface {
	CreateDialog(contact dto_discovery.Contact) (Dialog, error)
}

// Dialog represent established connection between 2 peers in network.
// Enables bidirectional communication with another peer.
type Dialog interface {
	Sender
	Receiver
	Close() error
}

// Receiver represents interface for:
//   - listening for asynchronous messages
//   - listening and serving HTTP-like requests
type Receiver interface {
	Receive(consumer MessageConsumer) error
	Respond(consumer RequestConsumer) error
}

// Sender represents interface for:
//   - sending asynchronous messages
//   - sending and HTTP-like request and waiting for response
type Sender interface {
	Send(producer MessageProducer) error
	Request(producer RequestProducer) (responsePtr interface{}, err error)
}
