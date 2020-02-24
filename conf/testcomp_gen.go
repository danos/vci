// Copyright (c) 2017-2019, AT&T Intellectual Property.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0
//
// This file provides utilities for creating a string version of a
// DotComponent file for testing.  It creates this independently of
// the INI file handler used by the main code so that it can be
// properly independent for test purposes.

package conf

import (
	"fmt"
	"strings"
)

type testModel struct {
	name             string
	modules          []string
	modelSets        []string
	checkOnlyImports []string
}

type TestComp struct {
	desc   string // Potentially mixed case, no spaces.
	name   string // Lower case version of description
	prefix string
	before []string
	after  []string
	models []testModel
	dflt   bool
}

const (
	// 'Base' is used rather than default to avoid confusion with the
	// 'default' component which has NO explicitly assigned modules and
	// instead owns all unowned modules.
	BaseModelSet           = "vyatta-v1"
	BaseNameAndModelPrefix = "net.vyatta.test"
	BaseModulePrefix       = "vyatta-test"
	BaseVersion            = "v1"
)

func CreateTestDotComponentFile(desc string) *TestComp {
	return &TestComp{
		desc:   desc,
		name:   strings.ToLower(desc),
		prefix: BaseNameAndModelPrefix,
	}
}

func (tc *TestComp) SetPrefix(prefix string) *TestComp {
	tc.prefix = prefix
	return tc
}

func (tc *TestComp) SetBefore(before ...string) *TestComp {
	if tc.prefix == "" {
		panic("Must set prefix prior to setting 'before' field.")
	}
	for ix, entry := range before {
		before[ix] = tc.prefix + "." + entry
	}
	tc.before = before
	return tc
}

func (tc *TestComp) SetAfter(after ...string) *TestComp {
	if tc.prefix == "" {
		panic("Must set prefix prior to setting 'after' field.")
	}
	for ix, entry := range after {
		after[ix] = tc.prefix + "." + entry
	}
	tc.after = after
	return tc
}

func (tc *TestComp) SetDefault() *TestComp {
	tc.dflt = true
	return tc
}

func (tc *TestComp) AddBaseModel() *TestComp {
	var tm = testModel{
		name: BaseNameAndModelPrefix + "." + tc.name,
		modules: []string{
			BaseModulePrefix + "-" + tc.name + "-" + BaseVersion},
		modelSets: []string{BaseModelSet},
	}
	tc.models = append(tc.models, tm)
	return tc
}

func (tc *TestComp) AddModel(
	name string,
	modules []string,
	modelSets []string,
) *TestComp {
	var tm = testModel{
		name:      name,
		modules:   modules,
		modelSets: modelSets,
	}
	tc.models = append(tc.models, tm)
	return tc
}

func (tc *TestComp) AddModelWithCheckImport(
	name string,
	modules []string,
	modelSets []string,
	checkOnlyImports []string,
) *TestComp {
	var tm = testModel{
		name:             name,
		modules:          modules,
		modelSets:        modelSets,
		checkOnlyImports: checkOnlyImports,
	}
	tc.models = append(tc.models, tm)
	return tc
}

func (tc *TestComp) ServiceName() string {
	return tc.prefix + "." + tc.name
}

func (tc *TestComp) String() string {
	componentStr := fmt.Sprintf(
		"[Vyatta Component]\n"+
			"Name=%s\n"+
			"Description=%s Test Component\n"+
			"ExecName=/opt/vyatta/sbin/%s-test\n"+
			"ConfigFile=/etc/vyatta/%s-test.conf\n",
		tc.ServiceName(),
		tc.desc,
		tc.name,
		tc.name)

	if len(tc.before) > 0 {
		componentStr += fmt.Sprintf("Before=%s\n", strings.Join(tc.before, ","))
	}
	if len(tc.after) > 0 {
		componentStr += fmt.Sprintf("After=%s\n", strings.Join(tc.after, ","))
	}

	if tc.dflt {
		componentStr += "DefaultComponent=true\n"
	}

	componentStr += "\n"

	for _, model := range tc.models {
		modelStr := fmt.Sprintf(
			"[Model %s]\n"+
				"Modules=%s\n"+
				"ModelSets=%s\n",
			model.name, strings.Join(model.modules, ","),
			strings.Join(model.modelSets, ","))
		componentStr += modelStr
		if len(model.checkOnlyImports) > 0 {
			componentStr += fmt.Sprintf("ImportsRequiredForCheck=%s\n",
				strings.Join(model.checkOnlyImports, ","))
		}
	}

	return componentStr
}
