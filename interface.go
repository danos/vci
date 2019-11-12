// Copyright (c) 2017-2019, AT&T Intellectual Property. All rights reserved.
// Copyright (c) 2016 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package vci

type PathError struct {
	Path    string
	Message string
}

func (e *PathError) Error() string {
	return e.Path + ":" + e.Message
}

// A Component represents an interface for a portion of the data-model.
// It maps the user accessible model of the system to the concrete
// model of the service that implements the functionallity.
type Component interface {
	// Model creates a new Model attached to this component.
	Model(model string) Model
	// Run attaches the component to the underlying transport
	// and begins the processing of messages destined for this
	// component.
	Run() error
	// Subscribe begins listening to a notification. When
	// a message is received it will be passed to the subscriber.
	// A subscribers is a function of any type. The notification
	// message will be unmarshalled into the type by the RFC7951
	// decoder.
	Subscribe(mod, name string, subscriber interface{}) error
	// Unsubscribe removes the listener for a particular notification.
	Unsubscribe(mod, name string) error
	// LookupSubscription will provide the Subscription representation
	// of a notification listener. This can be used to tweak the queuing
	// policies associated with the listener.
	LookupSubscription(mod, name string) *Subscription
	// Client provides access to the component's client representation
	// this is useful for sharing a client between the component and
	// the implementation so that the implementation can use the same
	// connection to the transport as the component.
	Client() *Client
	// Wait blocks until the component terminates its execution. If there was
	// an error during component execution it will be returned from Wait.
	Wait() error
	// Stop terminates the component execution. It will close all transport
	// connections and disconnect the component from the bus.
	Stop() error
}

// A Model represents a self-consistent set of YANG models. The Model
// can have configuration, operational data, and RPCs associated with
// it. The objects registered to receieve messages for the configuration,
// operational data, and RPCs implement the functionallity to map the
// user level data-model to the underlying system model.
type Model interface {
	// Config attaches a configuration handler to the model.
	// This handler must implement three methods:
	//   (1) Set(config T) error
	//       This method applies the configuration supplied to
	//       the underlying service.
	//   (2) Check(config T) error
	//       This method validates the configuration supplied
	//       against a set of constraints that cannot be modeled in YANG.
	//   (3) Get() (T, error)
	//       This method returns the configuration in a form that matches
	//       the data-model.
	// where T is any type that can be marshalled by the RFC7951
	// encoder.
	Config(object interface{}) Model
	// State attaches an operational state handler to the model.
	// This handler must implement one method:
	//   (1) Get() (T, error)
	//       This method returns the configuration in a form that matches
	//       the data-model.
	// where T is any type that can be marshalled by the RFC7951
	// encoder.
	State(object interface{}) Model
	// RPC attaches a set of RPCs to the model for a particular module name.
	// This handler must be one of the following forms:
	//   (1) A map from string to func(input T1) (output T2, error).
	//       In this form all methods are exposed with the names provided
	//       in the map.
	//   (2) An object with only methods of the following form:
	//       Name(input T1) (output T2, error)
	//       Names will be converted from the Go style CamelCase
	//       to the standard YANG convention of camel-case.
	// where T1, T2 are any types that can be marshalled by the
	// RFC7951 encoder.
	// The RPCs must implement the functionallity specified in the YANG model
	// and must conform to the model in both input and output.
	RPC(moduleName string, object interface{}) Model
}

// EmitNotification connects to the transport sends the notification
// then disconnects from the transport.
// If sending multiple notifications using a Client is a more efficient options.
func EmitNotification(
	moduleName, name string, val interface{},
) error {
	client, err := Dial()
	if err != nil {
		return err
	}
	defer client.Close()
	return client.Emit(moduleName, name, val)
}

// CallRPC calls the RPC corresponding to moduleName and rpcName
// using input as the input to the RPC.
// This provides a single shot RPC call without maintaining a client
// connection but it is more costly to call an RPC this way.
func CallRPC(
	moduleName, rpcName string,
	input interface{},
) *RPCCall {
	client, err := Dial()
	if err != nil {
		return &RPCCall{err: err}
	}
	defer client.Close()
	return client.Call(moduleName, rpcName, input)
}
