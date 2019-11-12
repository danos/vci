// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2016 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package test_helper

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

func MatchStringsUnordered(t *testing.T, description string, uexpect, uactual []string) {
	if len(uexpect) != len(uactual) {
		t.Errorf("Mismatching length for set of strings for %s:\n  expect %d, got %d",
			description, len(uexpect), len(uactual))
		return
	}
	var expect sort.StringSlice = uexpect
	expect.Sort()

	var actual sort.StringSlice = uactual
	actual.Sort()

	for i, v := range expect {
		desc := fmt.Sprintf("%s string %d", description, i)
		MatchString(t, desc, v, actual[i])
	}
}

func MatchStrings(t *testing.T, description string, expect, actual []string) {
	if len(expect) != len(actual) {
		t.Errorf("Mismatching length for set of strings for %s:\n  expect %d, got %d",
			description, len(expect), len(actual))
		return
	}
	for i, v := range expect {
		desc := fmt.Sprintf("%s string %d", description, i)
		MatchString(t, desc, v, actual[i])
	}
}

func MatchString(t *testing.T, description, expect, actual string) {
	if expect != actual {
		t.Errorf("%s doesn't match:\n  Expect: %s\n  Actual: %s",
			description, expect, actual)
	}
}

func CheckContains(t *testing.T, message, content string) {
	if !strings.Contains(message, content) {
		t.Fatalf("Expected to find '%s' in:\n%s", content, message)
	}
}
