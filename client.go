// Copyright (c) 2017-2019, 2021, AT&T Intellectual Property.
// All rights reserved.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package vci

import (
	"errors"
	"reflect"

	"github.com/danos/mgmterror"
)

var errNotImplemented = errors.New("not implemented")

// The Client supplies an encapsulated mechanism for
// performing operations on the VCI bus.
type Client struct {
	transport  transporter
	marshaller marshaller
	err        error
}

// Dial is the constructor for the default VCI client.
// It will return a client that can speak on the standard
// bus configuration.
func Dial() (*Client, error) {
	c := newClient().
		withTransport(defaultTransport()).
		dial()
	return c, c.checkConnection()
}

// NewClient will create a VCI client that is not connected to a Bus.
func newClient() *Client {
	return &Client{
		marshaller: defaultMarshaller(),
	}
}

// withTransport provides a mechanism for the client to attach to the Bus,
// this allows a transport to be shared between a component and a client.
// it is only used internally.
func (c *Client) withTransport(t transporter) *Client {
	c.transport = t
	return c
}

// dial establishes a connection to the bus.
func (c *Client) dial() *Client {
	if c.transport == nil {
		panic(errors.New(
			"Client.Dial called without providing a transport"))
	}
	c.err = c.transport.Dial()
	return c
}

// checkConnection will return an error
// if there was a problem connecting to the bus.
func (c *Client) checkConnection() error {
	return c.err
}

// Close closes the connection to VCI.
func (c *Client) Close() error {
	return c.transport.Close()
}

// Call will initiate a call to an RPC specified by the
// YANG module name and the RPC name. This will return
// a promise that can be fulfilled when the result is
// needed. The input object is marshalled using the RFC7951 encoder.
func (c *Client) Call(moduleName, rpcName string, input interface{}) *RPCCall {
	encodedData, err := c.marshalObject(input)
	if err != nil {
		return &RPCCall{err: err}
	}
	promise, err := c.transport.Call(moduleName, rpcName, encodedData)
	if err != nil {
		return &RPCCall{err: err}
	}
	return &RPCCall{client: c, promise: promise}
}

// Subscribe will allow one to subscribe to
// a notification specified by the YANG module name and
// the notification name. This takes a subscriber which may
// be one of:
//     (1) a function that takes any type as its input.
//         The notification will be unmarshalled into the type
//         using the RFC7951 decoder.
//     (2) a send channel of any type. The notification will be
//         unmarshalled into the type by the RFC7951 decoder.
// Any other type for a subscriber will signal an error on subscription.
func (c *Client) Subscribe(
	moduleName, notificationName string,
	subscriber interface{},
) *Subscription {
	wrapped, inputType, err := func(
		subscriber interface{},
	) (func(interface{}), reflect.Type, error) {
		val := reflect.ValueOf(subscriber)
		switch val.Kind() {
		case reflect.Func:
			if val.Type().NumIn() != 1 {
				return nil, nil,
					errors.New("Invalid subscriber type")
			}
			inputType := val.Type().In(0)
			return func(value interface{}) {
				val.Call([]reflect.Value{
					reflect.ValueOf(value)})
			}, inputType, nil
		case reflect.Chan:
			inputType := val.Type().Elem()
			return func(value interface{}) {
				val.Send(reflect.ValueOf(value))
			}, inputType, nil
		}
		return nil, nil, errors.New("Invalid subscriber type")
	}(subscriber)
	return newSubscription(c, moduleName, notificationName,
		wrapped, inputType, err)
}

// Emit will allow one to send to a notification
// specified by the YANG module name and the notification name.
// The supplied object will be marshalled using the RFC7951 encoder.
// If the object does not match the YANG schema for the notificaiton an
// error will be returned.
func (c *Client) Emit(
	moduleName, notificationName string,
	object interface{},
) error {
	encodedData, err := c.marshalObject(object)
	if err != nil {
		return err
	}
	return c.transport.Emit(moduleName, notificationName, encodedData)
}

// SetConfigForModel will set the configuration for the given model, using
// the model's component's registered Set method, marshallling the config
// from the provided object using the RFC7951 encoder.
func (c *Client) SetConfigForModel(
	modelName string,
	object interface{},
) error {
	encodedData, err := c.marshalObject(object)
	if err != nil {
		return err
	}
	return c.transport.SetConfigForModel(modelName, encodedData)
}

// CheckConfigForModel will validate the configuration for the given model,
// using the model's component's registered Check method, marshalling the
// config from the provided object using the RFC7951 encoder.
func (c *Client) CheckConfigForModel(
	modelName string,
	object interface{},
) error {
	encodedData, err := c.marshalObject(object)
	if err != nil {
		return err
	}
	return c.transport.CheckConfigForModel(modelName, encodedData)
}

// StoreConfigByModelInto will retrieve the configuration
// for a supplied model and unmarshal it into the supplied
// object using the RFC7951 decoder.
func (c *Client) StoreConfigByModelInto(
	modelName string,
	object interface{},
) error {
	var encodedData string
	err := c.transport.StoreConfigByModelInto(modelName, &encodedData)
	if err != nil {
		return err
	}
	return c.unmarshalObject(encodedData, object)
}

// StoreStateByModelInto will retrieve the operational state
// for a supplied model and unmarshal it into the supplied
// object using the RFC7951 decoder.
func (c *Client) StoreStateByModelInto(
	modelName string,
	object interface{},
) error {
	var encodedData string
	err := c.transport.StoreStateByModelInto(modelName, &encodedData)
	if err != nil {
		return err
	}
	return c.unmarshalObject(encodedData, object)
}

func (c *Client) marshalObject(object interface{}) (string, error) {
	if s, ok := object.(string); ok {
		return s, nil
	}
	buf, err := c.marshaller.Marshal(object)
	if err != nil {
		return "", mgmterror.NewMalformedMessageError()
	}
	return buf, nil
}

func (c *Client) unmarshalObject(encodedData string, object interface{}) error {
	if s, ok := object.(*string); ok {
		*s = encodedData
		return nil
	}
	err := c.marshaller.Unmarshal(encodedData, object)
	if err != nil {
		return mgmterror.NewMalformedMessageError()
	}
	return nil
}

// The RPCCall represents a promise to return the result of the
// RPC call upon request. RPCs happen asynchronously until the
// methods are called on this object.
type RPCCall struct {
	client  *Client
	err     error
	promise transportRPCPromise
}

// StoreOutputInto will unmarshal the output tree
// of the RPC into the supplied object using the RFC7951 decoder
// or an error if an error occurred during the call.
func (c *RPCCall) StoreOutputInto(object interface{}) error {
	if c.err != nil {
		return c.err
	}
	var encodedData string
	err := c.promise.StoreOutputInto(&encodedData)
	if err != nil {
		return err
	}
	return c.client.unmarshalObject(encodedData, object)
}
