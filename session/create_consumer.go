package session

import (
	"fmt"
	"github.com/mysterium/node/communication"
)

type SessionCreateConsumer struct {
	CurrentProposalId int
	SessionManager    ManagerInterface
}

func (consumer *SessionCreateConsumer) GetRequestEndpoint() communication.RequestEndpoint {
	return endpointSessionCreate
}

func (consumer *SessionCreateConsumer) NewRequest() (requestPtr interface{}) {
	var request SessionCreateRequest
	return &request
}

func (consumer *SessionCreateConsumer) Consume(requestPtr interface{}) (response interface{}, err error) {
	request := requestPtr.(*SessionCreateRequest)
	if consumer.CurrentProposalId != request.ProposalId {
		response = &SessionCreateResponse{
			Success: false,
			Message: fmt.Sprintf("Proposal doesn't exist: %d", request.ProposalId),
		}
		return
	}

	clientSession, err := consumer.SessionManager.Create()
	if err != nil {
		response = &SessionCreateResponse{
			Success: false,
			Message: "Failed to create session.",
		}
		return
	}

	response = &SessionCreateResponse{
		Success: true,
		Session: SessionDto{
			Id:     clientSession.Id,
			Config: clientSession.Config,
		},
	}
	return
}
