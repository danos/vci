// Copyright (c) 2017,2019, AT&T Intellectual Property. All rights reserved.
// Copyright (c) 2016 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package vci

import (
	"testing"
)

func TestGenYangName(t *testing.T) {
	names := map[string]string{
		"":       "",
		"fooB":   "foo-b",
		"fooBar": "foo-bar",
		"fooBAr": "foo-bar",
		"FOOBAR": "foobar",
		"foo1":   "foo1",
		"foo1B":  "foo-1b",
		"foo12B": "foo-12b",
		"FOO1B":  "foo-1b",
	}
	for inp, exp := range names {
		gen := genYangName(inp)
		if exp != gen {
			t.Errorf("Invalid name generated for [%v]; expected [%v], generated [%v]", inp, exp, gen)
		}
	}
}
