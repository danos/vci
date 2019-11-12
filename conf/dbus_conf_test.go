// Copyright (c) 2017-2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2016-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package conf

import (
	"fmt"
	"github.com/danos/vci/conf/test_helper"
	"github.com/go-ini/ini"
	"testing"
)

func checkServiceFile(t *testing.T, serviceFile, name, execName string) {
	iniFile, err := ini.Load([]byte(serviceFile))
	if err != nil {
		t.Fatalf("Unable to parse DBUS service file: %s", err.Error())
		return
	}

	checkSections(t, iniFile, "D-BUS Service")
	checkSectionKeys(t, iniFile, "D-BUS Service",
		"Name", "Notify", "Exec", "User", "SystemdService")
	checkSectionKeyEquals(t, iniFile, "D-BUS Service", "Name", name)
	checkSectionKeyEquals(t, iniFile, "D-BUS Service", "Notify", "true")
	checkSectionKeyEquals(t, iniFile, "D-BUS Service", "Exec", "/bin/systemctl start "+name)
	checkSectionKeyEquals(t, iniFile, "D-BUS Service", "User", "root")
	checkSectionKeyEquals(t, iniFile, "D-BUS Service", "SystemdService",
		name+".service")
}

func checkConfigFile(t *testing.T, configFile, name string) {
	test_helper.CheckContains(t, configFile, "<!DOCTYPE busconfig PUBLIC")
	test_helper.CheckContains(t, configFile,
		" \"-//freedesktop//DTD D-BUS Bus Configuration 1.0//EN\"")
	test_helper.CheckContains(t, configFile,
		" \"http://www.freedesktop.org/standards/dbus/1.0/busconfig.dtd\">")
	test_helper.CheckContains(t, configFile, "<busconfig>")
	test_helper.CheckContains(t, configFile, "	<policy user=\"root\">")
	test_helper.CheckContains(t, configFile,
		fmt.Sprintf("		<allow own=\"%s\"/>", name))
	test_helper.CheckContains(t, configFile,
		"		<allow send_destination=\"*\"/>")
	test_helper.CheckContains(t, configFile, "	</policy>")
	test_helper.CheckContains(t, configFile, "</busconfig>")
}

func verifyModelDbusServiceFile(
	t *testing.T,
	compConfig *ServiceConfig,
	modelName string,
) {
	model, ok := compConfig.ModelByName[modelName]
	if !ok {
		t.Fatalf("Component %s doesn't contain model %s\n",
			compConfig.Name, modelName)
	}
	serviceFile := string(model.GenerateDbusService())
	checkServiceFile(t, serviceFile, modelName, compConfig.ExecName)
}

func verifyModelDbusConfigFile(
	t *testing.T,
	compConfig *ServiceConfig,
	modelName string,
) {
	model, ok := compConfig.ModelByName[modelName]
	if !ok {
		t.Fatalf("Component %s doesn't contain model %s\n",
			compConfig.Name, modelName)
	}
	configFile := string(model.GenerateDbusConfig())
	checkConfigFile(t, configFile, modelName)
}

func getValidConfig(t *testing.T, test_config []byte) *ServiceConfig {
	compConfig, err := ParseConfiguration(test_config)
	if err != nil {
		t.Fatalf("Unexpected error when parsing config\n  %s", err.Error())
	}
	if compConfig == nil {
		t.Fatalf("Missing parsed configuration")
	}

	return compConfig
}

var dbusTestConfig []byte = []byte(`[Vyatta Component]
Name=net.vyatta.test.example
Description=Super Example Project
ExecName=systemctl start net.vyatta.test.example
ConfigFile=/etc/vyatta/example.conf

[Model net.vyatta.test.example]
Modules=example-v1,example-interfaces-v1
ModelSets=vyatta-v1,vyatta-v2

[Model org.ietf.test.example]
Modules=ietf-example
ModelSets=ietf-v1
`)

func TestCreateDbusServiceFileForComponent(t *testing.T) {
	compConfig := getValidConfig(t, dbusTestConfig)

	serviceFile := string(compConfig.GenerateDbusService())
	checkServiceFile(t, serviceFile, compConfig.Name, compConfig.ExecName)
}

func TestCreateDbusConfigFileForComponent(t *testing.T) {
	compConfig := getValidConfig(t, dbusTestConfig)

	configFile := string(compConfig.GenerateDbusConfig())
	checkConfigFile(t, configFile, compConfig.Name)
}

func TestCreateDbusServiceFileForModels(t *testing.T) {
	compConfig := getValidConfig(t, dbusTestConfig)

	verifyModelDbusServiceFile(t, compConfig,
		"net.vyatta.test.example")
	verifyModelDbusServiceFile(t, compConfig,
		"org.ietf.test.example")
}

func TestCreateDbusConfigFileForModels(t *testing.T) {
	compConfig := getValidConfig(t, dbusTestConfig)

	verifyModelDbusConfigFile(t, compConfig,
		"net.vyatta.test.example")
	verifyModelDbusConfigFile(t, compConfig,
		"org.ietf.test.example")
}
