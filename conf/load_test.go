// Copyright (c) 2017,2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2016 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package conf

import (
	"github.com/danos/vci/conf/test_helper"
	"testing"
)

func getComponentNames(comps []*ServiceConfig) []string {

	names := make([]string, len(comps))

	for i, c := range comps {
		names[i] = c.Name
	}

	return names
}

func TestLoadComponentDir(t *testing.T) {

	components, err := LoadComponentConfigDir("testdata")
	if err != nil {
		t.Fatalf("Failed to load test component directory: %s", err.Error())
	}

	if len(components) != 2 {
		t.Errorf("Unexpected number of components found:\n  expect: 2\n  actual: %d",
			len(components))
		t.Fatalf("  Found components: %s", getComponentNames(components))
	}

	expect := []string{
		"net.vyatta.test.service.test.a",
		"net.vyatta.test.service.test.b",
	}

	actual := getComponentNames(components)

	test_helper.MatchStringsUnordered(t, "Component names", expect, actual)
}
