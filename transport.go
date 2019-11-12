// Copyright (c) 2017-2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2016 by Brocade Communications Systems, Inc.
// All rights reserved
//
// SPDX-License-Identifier: MPL-2.0

package vci

const (
	yangdName       = "net.vyatta.vci.config.yangd.v1"
	yangdModuleName = "yangd-v1"
)

// The transportRPCPromise allows one to retrieve the value of an RPC call that
// was previously started.
type transportRPCPromise interface {
	StoreOutputInto(*string) error
}

// The transportSubscriber is a mechanism that will deliver a
// notification to a subscriber.
type transportSubscriber interface {
	Deliver(encodedData string) error
}

// The transportObject type represents any object that is to be exposed on
// the transport.
type transportObject interface {
	// Methods provides a set of methods that will be exposed on the transport.
	// Each method may only receieve a string and return an error or a pair of
	// string and error.
	Methods() map[string]interface{}
	// IsValid informs the transporter implementation if the object is valid or
	// if there was a problem when building it. If there was a problem an apporpriate
	// error should be returned.
	IsValid() bool
	// Name provides the name of this transport object and will be the name the
	// object is exposed on the transport as.
	Name() string
	// Type represnets the object type. This does not map one to one to a
	// go type. It is useful if the transport needs to expose objects of a
	// particular type differently than other objects. The current types are
	// "state", "config", and "rpc".
	Type() string
}

// The transporter interface represents an interface that can make appropriate
// calls on the underlying bus. The semantics for this interface are enforced
// by the testTransportSemantics unit tests. Any implementation should be
// validated against this test suite to ensure semantic compliance with the
// interface.
type transporter interface {
	// Dial connects to the underlying transport. It returns errors if
	// the transport is unavailable or if the connection fails for any
	// other reason.
	Dial() error
	// RequestIdentity registers a specific identification with the current
	// connection. This is optional but some may transports require this to
	// have addressing work properly.
	RequestIdentity(id string) error
	// Call calls an RPC on the transport. All information transmitted
	// on the transport is RFC7951 encoded strings.
	Call(moduleName, rpcName, input string) (transportRPCPromise, error)
	// Subscribe adds a subscirber for a given notification, the
	// transport must be able to support multiple subscribers for a
	// single notification name.
	Subscribe(moduleName, notificationName string,
		subscriber transportSubscriber) error
	// Unsubscribe removes a subscription to a notification. The subscription
	// is matched by the notification name and the subscriber.
	Unsubscribe(moduleName, notificationName string,
		subscriber transportSubscriber) error
	// Emit transmits a notification on the transport. An emitted notification
	// must be received by all subscribers, including subscribers on the
	// current connection.
	Emit(moduleName, notificationName, encodedData string) error
	// StoreConfigByModelInto will cause the configuration for a given
	// model to be queried and stored into the passed in pointer.
	StoreConfigByModelInto(modelName string, encodedData *string) error
	// StoreConfigByModelInto will cause the operational data for a given
	// model to be queried and stored into the passed in pointer.
	StoreStateByModelInto(modelName string, encodedData *string) error
	// Export will expose the transportObject on the transport so that
	// it may be accessed by Clients.
	Export(object transportObject) error
	// Close terminates the connection to the transport. After a close, no
	// notifications may be received nor can any calls be made.
	Close() error
}

// Allow default transport to be mocked out in tests
var defaultTransportConstructor func() transporter

func setDefaultTransportConstructor(constructor func() transporter) {
	defaultTransportConstructor = constructor
}

// defaultTransport constructs a connection to the default transport type
// for VCI Clients.
func defaultTransport() transporter {
	return defaultTransportConstructor()
}

// defaultMarshaller constructs a marshaller that uses the default encoding.
func defaultMarshaller() marshaller {
	return newRFC7951Marshaller()
}

// The Marshaller type is used to convert go objects to a string and from a
// string into an object.
type marshaller interface {
	Marshal(object interface{}) (string, error)
	Unmarshal(data string, object interface{}) error
	IsEmptyObject(data string) bool
}
