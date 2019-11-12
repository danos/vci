// Copyright (c) 2017,2019, AT&T Intellectual Property. All rights reserved.
// Copyright (c) 2016 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package vci

import (
	godbus "github.com/godbus/dbus"
	"testing"
)

func TestGenDBusName(t *testing.T) {
	names := map[string]string{
		"":        "",
		"f":       "F",
		"foo-b":   "FooB",
		"foo-bar": "FooBar",
		"foobar":  "Foobar",
		"foo1":    "Foo1",
		"foo1b":   "Foo1b",
		"foo-1b":  "Foo1b",
	}
	transport := newDBusTransport()
	for inp, exp := range names {
		gen := transport.convertYangNameToDBus(inp)
		if exp != gen {
			t.Errorf("Invalid name generated for [%v]; expected [%v], generated [%v]", inp, exp, gen)
		}
	}
}

//TestDBusRPCNameMappings verifies that RPC names and interfaces will be exposed
//on D-Bus using the proper names based on the module name.
func TestDBusRPCNameMappings(t *testing.T) {
	tport := newDBusTransport()
	if tport.getModuleRPCObjectPath("test-v1") !=
		godbus.ObjectPath("/test_v1/rpc") {
		t.Fatal("RPC Object Path is incorrect")
	}
	if tport.getModuleRPCInterfaceName("test-v1") !=
		"yang.module.TestV1.RPC" {
		t.Fatal("RPC Interface is incorrect")
	}
	if tport.getModuleRPCObjectPath("test-mod1-v1") !=
		godbus.ObjectPath("/test_mod1_v1/rpc") {
		t.Fatal("RPC Object Path is incorrect")
	}
	if tport.getModuleRPCInterfaceName("test-mod1-v1") !=
		"yang.module.TestMod1V1.RPC" {
		t.Fatal("RPC Interface is incorrect")
	}
}

func TestDBusTransportSemantics(t *testing.T) {
	setDefaultTransportConstructor(func() transporter {
		return newDBusSessionTransport()
	})
	testTransportSemantics(t, newDBusSessionTransport())
}
