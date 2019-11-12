// Copyright (c) 2017-2019, AT&T Intellectual Property. All rights reserved.
// Copyright (c) 2016 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package vci

import (
	"sync"
)

type model struct {
	name      string
	component *component
	transport transporter
	client    *Client

	cfg   *config
	state *state
	rpcs  map[string]*rpcObject
}

func newModel(name string, c *component) *model {
	return newModelWithTransport(name, c, defaultTransport())
}

func newModelWithTransport(name string, c *component, transport transporter) *model {
	m := &model{
		name:      name,
		component: c,
		rpcs:      make(map[string]*rpcObject),
		client:    newClient(),
	}
	if transport == nil {
		transport = defaultTransport()
	}
	m.withTransport(transport)
	return m
}

func (m *model) Config(cfg interface{}) Model {
	m.cfg = newConfig(cfg, m.client)
	return m
}

func (m *model) State(state interface{}) Model {
	m.state = newState(state, m.client)
	return m
}

func (m *model) RPC(moduleName string, rpc interface{}) Model {
	m.rpcs[moduleName] = newRPC(moduleName, rpc, m.client)
	return m
}

func (m *model) withTransport(t transporter) *model {
	m.transport = t
	if m.client != nil {
		m.client.withTransport(t)
	}
	return m
}

func (m *model) run() error {
	err := m.transport.Dial()
	if err != nil {
		return err
	}
	err = m.register()
	if err != nil {
		return err
	}
	m.component.wg.Add(1)
	return m.transport.RequestIdentity(m.name)
}

func (m *model) register() error {
	if m.cfg != nil {
		err := m.transport.Export(m.cfg)
		if err != nil {
			return err
		}
	}
	if m.state != nil {
		err := m.transport.Export(m.state)
		if err != nil {
			return err
		}
	}
	for _, rpc := range m.rpcs {
		err := m.transport.Export(rpc)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *model) stop() error {
	m.component.wg.Done()
	return m.transport.Close()
}

type component struct {
	name      string
	models    []*model
	transport transporter
	client    *Client
	wg        sync.WaitGroup

	subscriptions struct {
		mu             sync.RWMutex
		runOnSubscribe bool
		subs           map[string]*Subscription
	}
}

// NewComponent creates a concrete representation of the Component
// interface and returns the opaque Component so it may be used.
// Refer to the Component interface documentation for more information.
func NewComponent(name string) Component {
	comp := &component{name: name, client: newClient()}
	comp.subscriptions.subs = make(map[string]*Subscription)
	comp.withTransport(defaultTransport())
	return comp
}

func (c *component) withTransport(transport transporter) *component {
	c.transport = transport
	if c.client != nil {
		c.client.withTransport(c.transport)
	}
	for _, model := range c.models {
		model.withTransport(c.transport)
	}
	return c
}

func (c *component) Model(name string) Model {
	newModel := newModelWithTransport(name, c, c.transport)
	c.models = append(c.models, newModel)
	return newModel
}

func (c *component) Unsubscribe(mod, name string) error {
	c.subscriptions.mu.Lock()
	defer c.subscriptions.mu.Unlock()
	sname := mod + "/" + name
	s := c.subscriptions.subs[sname]
	delete(c.subscriptions.subs, sname)
	return s.Cancel()
}

func (c *component) Subscribe(mod, name string, subscriber interface{}) error {
	c.subscriptions.mu.Lock()
	defer c.subscriptions.mu.Unlock()
	s := c.client.Subscribe(mod, name, subscriber)
	subName := mod + "/" + name
	c.subscriptions.subs[subName] = s
	if c.subscriptions.runOnSubscribe {
		return s.Run()
	}
	return nil
}

func (c *component) LookupSubscription(mod, name string) *Subscription {
	c.subscriptions.mu.RLock()
	defer c.subscriptions.mu.RUnlock()
	subName := mod + "/" + name
	return c.subscriptions.subs[subName]
}

func (c *component) Run() error {
	err := c.transport.Dial()
	if err != nil {
		return err
	}
	err = c.register()
	if err != nil {
		return err
	}
	c.wg.Add(1)
	return c.transport.RequestIdentity(c.name)
}

func (c *component) Client() *Client {
	c.transport.Dial()
	return c.client
}

func (c *component) Wait() error {
	c.wg.Wait()
	return nil
}

func (c *component) Stop() error {
	for _, model := range c.models {
		_ = model.stop()
	}
	c.wg.Done()
	return c.transport.Close()
}

func (c *component) register() error {
	for _, model := range c.models {
		err := model.run()
		if err != nil {
			return err
		}
	}

	c.subscriptions.mu.Lock()
	for _, sub := range c.subscriptions.subs {
		err := sub.Run()
		if err != nil {
			return err
		}
	}
	c.subscriptions.runOnSubscribe = true
	c.subscriptions.mu.Unlock()
	return nil
}
