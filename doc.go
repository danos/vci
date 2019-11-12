// Package vci implements an interface to the Vyatta Component Infrastructure.
// VCI is a common interface to YANG modeled portions of the system. VCI
// provides a model concept that allow for implementing multiple YANG models
// by a single component. A model is a group of YANG modules that are
// considered to be one of the component's data-models. A component may need
// more than one module to effectively do it's job, the model concept is used
// to group these modules together so when new configration is sent to the
// component it receives all the information it requires. See the docs
// directory for more detailed information on the general operation of VCI.
//
// The package consists of a few parts:
//     (1) Components that are interfaces to portions of the system
//         that can be accessed by VCI clients.
//     (2) Client that provides a conveinent library for talking to components.
//     (3) Subscriptions that provide a mechanism for controlling the receipt of
//         notifications.
//     (4) A few error types that allow one to create errors that conform to
//         RFC6020's described errors.
package vci
