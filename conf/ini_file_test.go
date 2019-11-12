// Copyright (c) 2018-2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2016-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package conf

import (
	"github.com/go-ini/ini"
	"strings"
	"testing"
)

// Expect exact match.  DEFAULT is added by iniFile parser, so we silently
// add here to match.
func checkSections(t *testing.T, iniFile *ini.File, sectionNames ...string) {
	actualSections := iniFile.SectionStrings()
	sectionNames = append(sectionNames, "DEFAULT")
	if len(actualSections) != len(sectionNames) {
		t.Fatalf("Unexpected number of sections.\nGot:\t%v\nExp:\t%v",
			actualSections, sectionNames)
		return
	}
	for _, expName := range sectionNames {
		found := false
		for _, actualName := range actualSections {
			if expName == actualName {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Unable to find section [%s] in ini file", expName)
			return
		}
	}
}

// Assumes section exists
func checkSectionKeys(
	t *testing.T,
	iniFile *ini.File,
	sectionName string,
	keyNames ...string,
) {
	actualKeys := iniFile.Section(sectionName).KeyStrings()
	if len(actualKeys) != len(keyNames) {
		t.Fatalf("Unexpected number of keys.\nGot:\t%v\nExp:\t%v",
			actualKeys, keyNames)
		return
	}
	for _, expKey := range keyNames {
		found := false
		for _, actualKey := range actualKeys {
			if expKey == actualKey {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Unable to find key '%s' in [%s] section",
				expKey, sectionName)
			return
		}
	}
}

func checkSectionKeysNotPresent(
	t *testing.T,
	iniFile *ini.File,
	sectionName string,
	keyNames ...string,
) {
	actualKeys := iniFile.Section(sectionName).KeyStrings()
	for _, unexpKey := range keyNames {
		for _, actualKey := range actualKeys {
			if unexpKey == actualKey {
				t.Fatalf("Found unexpected key '%s' in [%s] section",
					unexpKey, sectionName)
				return
			}
		}
	}
}

func checkSectionKeyEquals(
	t *testing.T,
	iniFile *ini.File,
	sectionName, keyName, expValue string,
) {
	section, err := iniFile.GetSection(sectionName)
	if err != nil {
		t.Fatalf("Unable to find [%s] section in file.", sectionName)
		return
	}

	key, err := section.GetKey(keyName)
	if err != nil {
		t.Fatalf("Unable to find key '%s' in unit [%s]", keyName, sectionName)
		return
	}

	if expValue != key.Value() {
		t.Logf("Checking [%s] %s='\n", sectionName, keyName)
		t.Fatalf("Expected:\t'%s'\nActual:\t%s", expValue, key.Value())
		return
	}
}

func checkSectionKeyContains(
	t *testing.T,
	iniFile *ini.File,
	sectionName, keyName, expSubstring string,
) {
	section, err := iniFile.GetSection(sectionName)
	if err != nil {
		t.Fatalf("Unable to find [%s] section in file.", sectionName)
		return
	}

	key, err := section.GetKey(keyName)
	if err != nil {
		t.Fatalf("Unable to find key '%s' in unit [%s]", keyName, sectionName)
		return
	}

	if !strings.Contains(key.Value(), expSubstring) {
		t.Fatalf("Expected substr:\t'%s'\nActual string:\t%s",
			expSubstring, key.Value())
		return
	}
}
