// Copyright (c) 2017,2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2016 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"github.com/danos/vci/conf"
	"github.com/danos/vci/conf/test_helper"
	"testing"
)

var test_config []byte = []byte(`[Vyatta Component]
Name=net.vyatta.test.example
Description=Super Example Project
ExecName=/opt/vyatta/sbin/example-service
ConfigFile=/etc/vyatta/example.conf

[Model net.vyatta.test.example]
Modules=example-v1,example-interfaces-v1
ModelSets=vyatta-v1,vyatta-v2

[Model org.ietf.test.example]
Modules=ietf-example
ModelSets=ietf-v1
`)

func TestFileNames(t *testing.T) {
	config, err := conf.ParseConfiguration(test_config)
	if err != nil {
		t.Fatalf("Unexpected error when parsing config\n  %s", err.Error())
	}
	if config == nil {
		t.Fatalf("Missing parsed configuration")
	}

	expect := []string{
		//  Main Component
		"/lib/systemd/system/vyatta-service-example.service",
		"/lib/systemd/system/net.vyatta.test.example.service",
		"/usr/share/dbus-1/system-services/net.vyatta.test.example.service",
		"/etc/vci/bus.d/net.vyatta.test.example.conf",

		// Model brocade
		"/lib/systemd/system/net.vyatta.test.example.service",
		"/usr/share/dbus-1/system-services/net.vyatta.test.example.service",
		"/etc/vci/bus.d/net.vyatta.test.example.conf",

		// Model ietf
		"/lib/systemd/system/org.ietf.test.example.service",
		"/usr/share/dbus-1/system-services/org.ietf.test.example.service",
		"/etc/vci/bus.d/org.ietf.test.example.conf",
	}
	files := getFiles("vyatta-service-example", config)
	actual := []string{}
	for _, f := range files {
		switch file := f.(type) {
		case *configFile:
			actual = append(actual, file.name)
		case *symlink:
			actual = append(actual, file.name)
		}
	}

	// Due to the hash map used to store the models, the ordering can vary
	test_helper.MatchStringsUnordered(t, "Configuration File Names", expect, actual)
}
