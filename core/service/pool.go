/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package service

import (
	"sync"

	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/core/policy"
	"github.com/mysteriumnetwork/node/core/service/servicestate"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/utils"
	"github.com/pkg/errors"
)

// ID represent unique identifier of the running service.
type ID string

// RunnableService represents a runnable service
type RunnableService interface {
	Stop() error
}

// Pool is responsible for supervising running instances
type Pool struct {
	eventPublisher Publisher
	instances      map[ID]*Instance
	sync.Mutex
}

// Publisher is responsible for publishing given events
type Publisher interface {
	Publish(topic string, data interface{})
}

// NewPool returns a empty service pool
func NewPool(eventPublisher Publisher) *Pool {
	return &Pool{
		eventPublisher: eventPublisher,
		instances:      make(map[ID]*Instance),
	}
}

// Add registers a service to running instances pool
func (p *Pool) Add(instance *Instance) {
	p.Lock()
	defer p.Unlock()

	p.instances[instance.id] = instance
}

// Del removes a service from running instances pool
func (p *Pool) Del(id ID) {
	p.Lock()
	defer p.Unlock()
	p.del(id)
}

func (p *Pool) del(id ID) {
	delete(p.instances, id)
}

// ErrNoSuchInstance represents the error when we're stopping an instance that does not exist
var ErrNoSuchInstance = errors.New("no such instance")

// Stop kills all sub-resources of instance
func (p *Pool) Stop(id ID) error {
	p.Lock()
	defer p.Unlock()
	return p.stop(id)
}

func (p *Pool) stop(id ID) error {
	instance, ok := p.instances[id]
	if !ok {
		return ErrNoSuchInstance
	}

	errStop := utils.ErrorCollection{}
	if instance.discovery != nil {
		instance.discovery.Stop()
	}
	if instance.dialogWaiter != nil {
		errStop.Add(instance.dialogWaiter.Stop())
	}
	if instance.service != nil {
		errStop.Add(instance.service.Stop())
	}

	p.del(id)

	instance.setState(servicestate.NotRunning)

	return errStop.Errorf("ErrorCollection(%s)", ", ")
}

// StopAll kills all running instances
func (p *Pool) StopAll() error {
	p.Lock()
	defer p.Unlock()
	errStop := utils.ErrorCollection{}
	for id := range p.instances {
		errStop.Add(p.stop(id))
	}

	return errStop.Errorf("Some instances did not stop: %v", ". ")
}

// List returns all running service instances.
func (p *Pool) List() map[ID]*Instance {
	p.Lock()
	defer p.Unlock()
	return p.instances
}

// Instance returns service instance by the requested id.
func (p *Pool) Instance(id ID) *Instance {
	p.Lock()
	defer p.Unlock()
	return p.instances[id]
}

// NewInstance creates new instance of the service.
func NewInstance(
	options Options,
	state servicestate.State,
	service RunnableService,
	proposal market.ServiceProposal,
	policies *policy.Repository,
	dialog communication.DialogWaiter,
	discovery Discovery,
) *Instance {
	return &Instance{
		options:      options,
		state:        state,
		service:      service,
		proposal:     proposal,
		policies:     policies,
		dialogWaiter: dialog,
		discovery:    discovery,
	}
}

// Instance represents a run service
type Instance struct {
	id             ID
	state          servicestate.State
	options        Options
	service        RunnableService
	proposal       market.ServiceProposal
	policies       *policy.Repository
	dialogWaiter   communication.DialogWaiter
	discovery      Discovery
	eventPublisher Publisher

	stateLock sync.RWMutex
}

// Options returns options used to start service
func (i *Instance) Options() Options {
	return i.options
}

// Proposal returns service proposal of the running service instance.
func (i *Instance) Proposal() market.ServiceProposal {
	return i.proposal
}

// Policies returns service policies of the running service instance.
func (i *Instance) Policies() *policy.Repository {
	return i.policies
}

// State returns the service instance state.
func (i *Instance) State() servicestate.State {
	i.stateLock.RLock()
	defer i.stateLock.RUnlock()
	return i.state
}

func (i *Instance) setState(newState servicestate.State) {
	i.stateLock.Lock()
	defer i.stateLock.Unlock()
	i.state = newState

	i.eventPublisher.Publish(servicestate.AppTopicServiceStatus, i.toEvent())
}

// toEvent returns an event representation of the instance
func (i *Instance) toEvent() servicestate.AppEventServiceStatus {
	return servicestate.AppEventServiceStatus{
		ID:         string(i.id),
		ProviderID: i.proposal.ProviderID,
		Type:       i.proposal.ServiceType,
		Status:     string(i.state),
	}
}
