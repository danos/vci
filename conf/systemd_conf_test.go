// Copyright (c) 2017-2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2016-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package conf

import (
	"fmt"
	"strings"
	"testing"

	"github.com/go-ini/ini"
)

var systemdTestConfig []byte = []byte(`[Vyatta Component]
Name=net.vyatta.test.example
Description=Super Example Project
ExecName=/opt/vyatta/sbin/example-service
ConfigFile=/etc/vyatta/example.conf
Before=compB1,compB2,compB3
After=compA1,compA2

[Model net.vyatta.test.example]
Modules=example-v1,example-interfaces-v1
ModelSets=vyatta-v1,vyatta-v2

[Model org.ietf.test.example]
Modules=ietf-example
ModelSets=ietf-v1
`)

func TestCreateSystemdServiceFile(t *testing.T) {
	compConfig := getValidConfig(t, systemdTestConfig)
	systemdServiceFile := compConfig.GenerateSystemdService()
	iniFile, err := ini.Load(systemdServiceFile)
	if err != nil {
		t.Fatalf("Unable to parse systemd service file: %s", err.Error())
		return
	}

	checkSections(t, iniFile, "Unit", "Service", "Install")

	checkSectionKeys(t, iniFile, "Unit", "Description", "Before", "After", "BindsTo")
	checkSectionKeyEquals(t, iniFile, "Unit", "Description",
		compConfig.Description)
	checkSectionKeyEquals(t, iniFile, "Unit", "Before",
		strings.Join(compConfig.Before, " "))
	checkSectionKeyEquals(t, iniFile, "Unit", "After",
		strings.Join(compConfig.After, " "))

	checkSectionKeys(t, iniFile, "Service", "Type", "Restart", "ExecStart")
	checkSectionKeyEquals(t, iniFile, "Service", "Type", "notify")
	checkSectionKeyEquals(t, iniFile, "Service", "Restart", "on-failure")
	checkSectionKeyEquals(t, iniFile, "Service", "ExecStart",
		compConfig.ExecName)

	checkSectionKeys(t, iniFile, "Install", "Alias")
	checkSectionKeyContains(t, iniFile, "Install", "Alias",
		fmt.Sprintf("%s.service", compConfig.Name))
	for mod_name, _ := range compConfig.ModelByName {
		checkSectionKeyContains(t, iniFile, "Install", "Alias",
			mod_name+".service")
	}
}

var systemdTestConfigNoBeforeOrAfter []byte = []byte(`[Vyatta Component]
Name=com.brocade.vyatta.example
Description=Super Example Project
ExecName=/opt/vyatta/sbin/example-service
ConfigFile=/etc/vyatta/example.conf

[Model com.brocade.vyatta.example.brocade]
Modules=example-v1,example-interfaces-v1
ModelSets=brocade-v1
`)

func TestSystemdFileNoBeforeAfterWantedByKeys(t *testing.T) {
	compConfig := getValidConfig(t, systemdTestConfigNoBeforeOrAfter)
	systemdServiceFile := compConfig.GenerateSystemdService()
	iniFile, err := ini.Load(systemdServiceFile)
	if err != nil {
		t.Fatalf("Unable to parse systemd service file: %s", err.Error())
		return
	}

	checkSectionKeysNotPresent(t, iniFile, "Unit",
		"Before", "WantedBy")
	checkSectionKeyEquals(t, iniFile, "Unit", "After",
		"vyatta-vci-bus.service")
}

var systemdTestConfigStartOnBoot []byte = []byte(`[Vyatta Component]
Name=com.brocade.vyatta.example
Description=Super Example Project
ExecName=/opt/vyatta/sbin/example-service
ConfigFile=/etc/vyatta/example.conf
StartOnBoot=true

[Model com.brocade.vyatta.example.brocade]
Modules=example-v1,example-interfaces-v1
ModelSets=brocade-v1
`)

func TestSystemdFileStartOnBoot(t *testing.T) {
	compConfig := getValidConfig(t, systemdTestConfigStartOnBoot)
	systemdServiceFile := compConfig.GenerateSystemdService()
	iniFile, err := ini.Load(systemdServiceFile)
	if err != nil {
		t.Fatalf("Unable to parse systemd service file: %s", err.Error())
		return
	}

	checkSectionKeyEquals(t, iniFile, "Install", "WantedBy", "multi-user.target")
}
