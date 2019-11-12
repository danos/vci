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
	"strings"
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

func compareCSVs(t *testing.T, desc string, actual, expected []string) {
	if len(actual) != len(expected) {
		t.Logf("%s: number of actual entries (%d) != expected (%d)\n",
			desc, len(actual), len(expected))
		t.Fatalf("Act: %v\nExp: %v\n", actual, expected)
		return
	}

	for index, actEntry := range actual {
		if actEntry != expected[index] {
			t.Logf("%s: mismatched entry (index %d)\n", desc, index)
			t.Fatalf("Act: %v\nExp: %v\n", actual, expected)
			return
		}
	}
}

func TestReadInputSuccess(t *testing.T) {
	config, err := ParseConfiguration(test_config)
	if err != nil {
		t.Fatalf("Unexpected error when parsing config\n  %s", err.Error())
	}
	if config == nil {
		t.Fatalf("Missing parsed configuration")
	}

	expectDesc := "Super Example Project"
	expectBus := "net.vyatta.test.example"
	expectExec := "/opt/vyatta/sbin/example-service"
	expectCfg := []string{"/etc/vyatta/example.conf"}

	test_helper.MatchString(t, "Description field", expectDesc, config.Description)
	test_helper.MatchString(t, "Name field", expectBus, config.Name)
	test_helper.MatchString(t, "ExecName field", expectExec, config.ExecName)
	compareCSVs(t, "ConfigFile field", config.ConfigFiles, expectCfg)
}

func TestMissingField(t *testing.T) {
	mandatoryFields := []string{
		"Description", "Name", "ExecName", "ConfigFile",
	}

	for _, missing := range mandatoryFields {

		bad_config := "[Vyatta Component]\n"
		for _, field := range mandatoryFields {
			if field != missing {
				bad_config += fmt.Sprintf("%s=SomeValue\n", field)
			}
		}

		_, err := ParseConfiguration([]byte(bad_config))
		if err == nil {
			t.Fatalf("Unexpected success with missing field %s", missing)
		}

		if _, ok := err.(MissingFieldError); !ok {
			t.Fatalf("Unexpected error type with missing field\n  %s", err.Error())
		}

		expect := fmt.Sprintf("Missing %s field from Vyatta Component section", missing)
		test_helper.MatchString(t, "Missing Field Error", expect, err.Error())
	}
}

// 2 models, same name, same component
const testComp_twoModelsSameName = `[Vyatta Component]
Name=net.vyatta.test.service.test
Description=Test Component
ExecName=/opt/vyatta/sbin/test-service
ConfigFile=/etc/vyatta/test.conf

[Model net.vyatta.test.service.test.a]
Modules=vyatta-service-test-a-v1
ModelSets=vyatta-v1

[Model net.vyatta.test.service.test.a]
Modules=vyatta-service-test-b-v1
ModelSets=open-v1`

func TestDuplicateSection(t *testing.T) {
	_, err := ParseConfiguration([]byte(testComp_twoModelsSameName))
	if err != nil {
		test_helper.CheckContains(t, err.Error(), "Duplicate section")
		test_helper.CheckContains(t, err.Error(),
			"Model net.vyatta.test.service.test.a")
	} else {
		t.Fatalf("Duplicate section should have been detected.")
	}
}

const testComp_CompNameWithDotService = `[Vyatta Component]
Name=net.vyatta.test.service
Description=Test Component
ExecName=/opt/vyatta/sbin/test-service
ConfigFile=/etc/vyatta/test.conf

[Model net.vyatta.test.service.test.a]
Modules=vyatta-service-test-a-v1
ModelSets=vyatta-v1`

func TestComponentNameIncludesDotService(t *testing.T) {
	_, err := ParseConfiguration([]byte(testComp_CompNameWithDotService))
	if err != nil {
		test_helper.CheckContains(t, err.Error(),
			"Component Name must not include '.service'")
	} else {
		t.Fatalf("Component name with '.service' suffix not detected.")
	}
}

// Badly formatted but salvageable 'Before' and 'After' statements.
const testComp_salvageableDeps = `[Vyatta Component]
Name=net.vyatta.test.service.test
Description=Test Component
ExecName=/opt/vyatta/sbin/test-service
ConfigFile=/etc/vyatta/test.conf
Before=Components.service, With,Trailing.service , Comma,
After= MoreComponents, TrailingSpace.service, 

[Model net.vyatta.test.service.test.a]
Modules=vyatta-service-test-b-v1
ModelSets=open-v1`

func TestSalvageableDeps(t *testing.T) {
	ms, err := ParseConfiguration([]byte(testComp_salvageableDeps))
	if err != nil {
		t.Fatalf("Spaces and commas should have been dealt with.")
		return
	}
	before := ms.Before
	expBefore := []string{
		"Components.service",
		"With.service",
		"Trailing.service",
		"Comma.service",
	}
	compareCSVs(t, "Salvageable Before stmt", before, expBefore)

	after := ms.After
	expAfter := []string{
		"MoreComponents.service",
		"TrailingSpace.service",
	}
	compareCSVs(t, "Salvageable After stmt", after, expAfter)
}

const testComp_unsalvageableDeps = `[Vyatta Component]
Name=net.vyatta.test.service.test
Description=Test Component
ExecName=/opt/vyatta/sbin/test-service
ConfigFile=/etc/vyatta/test.conf
Before=Component With Spaces

[Model net.vyatta.test.service.test.a]
Modules=vyatta-service-test-b-v1
ModelSets=open-v1`

func TestUnsalvageableDeps(t *testing.T) {
	_, err := ParseConfiguration([]byte(testComp_unsalvageableDeps))
	if err != nil {
		test_helper.CheckContains(t, err.Error(),
			"Unable to parse 'Before': 'Component With Spaces'")
		test_helper.CheckContains(t, err.Error(),
			"Entries may not contain spaces")
	} else {
		t.Fatalf("Parsing should have failed.")
		return
	}
}

// Badly formatted but salvageable ConfigFile statement.
const testComp_salvageableCfgFiles = `[Vyatta Component]
Name=net.vyatta.test.service.test
Description=Test Component
ExecName=/opt/vyatta/sbin/test-service
ConfigFile=first.file, secondWithSpaceBeforeAndAfter ,thirdTrailing.WS 

[Model net.vyatta.test.service.test.a]
Modules=vyatta-service-test-b-v1
ModelSets=open-v1`

func TestSalvageableCfgFiles(t *testing.T) {
	ms, err := ParseConfiguration([]byte(testComp_salvageableCfgFiles))
	if err != nil {
		t.Fatalf("Spaces and commas should have been dealt with.")
		return
	}
	cfgFiles := ms.ConfigFiles
	expCfgFiles := []string{
		"first.file",
		"secondWithSpaceBeforeAndAfter",
		"thirdTrailing.WS",
	}
	compareCSVs(t, "Salvageable Config Files", cfgFiles, expCfgFiles)
}

const testComp_parseDefault = `[Vyatta Component]
Name=net.vyatta.test.service.test
Description=Test Component
ExecName=/opt/vyatta/sbin/test-service
ConfigFile=/etc/vyatta/test.conf
%s

[Model net.vyatta.test.service.test.a]
Modules=vyatta-service-test-b-v1
ModelSets=open-v1`

func TestParseDefaultComponentTrue(t *testing.T) {
	dotCompFile := fmt.Sprintf(testComp_parseDefault, "DefaultComponent=true")
	svcCfg, err := ParseConfiguration([]byte(dotCompFile))
	if err != nil {
		t.Fatalf("Unable to parse configuration: %s\n", err.Error())
		return
	}
	if svcCfg.DefaultComp != true {
		t.Fatalf("DefaultComponent not set to true!")
		return
	}
}

func TestParseDefaultComponentFaLsE(t *testing.T) {
	dotCompFile := fmt.Sprintf(testComp_parseDefault, "DefaultComponent=FaLsE")
	svcCfg, err := ParseConfiguration([]byte(dotCompFile))
	if err != nil {
		t.Fatalf("Unable to parse configuration: %s\n", err.Error())
		return
	}
	if svcCfg.DefaultComp != false {
		t.Fatalf("DefaultComponent not set to false!")
		return
	}
}

func TestParseDefaultComponentInvalid(t *testing.T) {
	dotCompFile := fmt.Sprintf(testComp_parseDefault, "DefaultComponent=foo")
	_, err := ParseConfiguration([]byte(dotCompFile))
	if err == nil {
		t.Fatalf("Default component should not have been parsed.")
		return
	}
	if !strings.Contains(err.Error(), "Value 'foo' must be 'true' or 'false'") {
		t.Fatalf("DefaultComponent=foo: wrong error '%s'", err.Error())
		return
	}
}

func TestParseNoDefaultComponent(t *testing.T) {
	dotCompFile := fmt.Sprintf(testComp_parseDefault, "")
	svcCfg, err := ParseConfiguration([]byte(dotCompFile))
	if err != nil {
		t.Fatalf("Unable to parse configuration: %s\n", err.Error())
		return
	}
	if svcCfg.DefaultComp != false {
		t.Fatalf("DefaultComponent not set to false!")
		return
	}
}

type expectedBus struct {
	name      string
	modules   []string
	modelsets []string
}

func (expect *expectedBus) match(t *testing.T, c *ServiceConfig) {

	actual := c.ModelByName[expect.name]
	if actual == nil {
		t.Errorf("Failed to find bus %s", expect.name)
		return
	}
	test_helper.MatchString(t, "bus name", expect.name, actual.Name)
	test_helper.MatchStrings(t, "bus modules", expect.modules, actual.Modules)
	test_helper.MatchStrings(t, "bus model sets", expect.modelsets,
		actual.ModelSets)

	for _, m := range expect.modelsets {
		bus := c.ModelByModelSet[m]
		// Check that the found bus matches the one stored for the model set
		if bus == nil {
			t.Errorf("Missing bus for model %s", m)
		} else if bus != actual {
			t.Errorf("Unexpected bus found for model %s: %s", m, bus.Name)
		}
	}
}

func TestParseModels(t *testing.T) {
	config, err := ParseConfiguration(test_config)
	if err != nil {
		t.Fatalf("Unexpected error when parsing config:\n  %s", err.Error())
	}
	if config == nil {
		t.Fatalf("Missing parsed configuration")
	}

	expectBus1 := &expectedBus{
		"net.vyatta.test.example",
		[]string{"example-v1", "example-interfaces-v1"},
		[]string{"vyatta-v1", "vyatta-v2"},
	}
	expectBus1.match(t, config)

	expectBus2 := &expectedBus{
		"org.ietf.test.example",
		[]string{"ietf-example"},
		[]string{"ietf-v1"},
	}
	expectBus2.match(t, config)
}
