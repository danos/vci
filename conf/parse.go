// Copyright (c) 2017-2020, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2016-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package conf

import (
	"fmt"
	"strings"

	"github.com/go-ini/ini"
)

// 'ini' file merges duplicate sections, taking last value assigned to any
// field.  This is likely to lead to unexpected behaviour so is detected and
// reported.
func checkForDuplicateSections(iniFile string) error {
	lines := strings.Split(iniFile, "\n")
	sectionMap := make(map[string]bool)

	for _, line := range lines {
		if strings.HasPrefix(line, "[") {
			if _, ok := sectionMap[line]; ok {
				return fmt.Errorf("Duplicate section: %s\n", line)
			}
			sectionMap[line] = true
		}
	}

	return nil
}

func ParseConfiguration(input []byte) (*ServiceConfig, error) {
	if err := checkForDuplicateSections(string(input)); err != nil {
		return nil, err
	}

	iniFile, err := ini.Load(input)
	if err != nil {
		return nil, err
	}

	config := &ServiceConfig{
		ModelByName:     make(map[string]*Model),
		ModelByModelSet: make(map[string]*Model),
	}

	const busPrefix = "Model "
	for _, section := range iniFile.Sections() {
		name := section.Name()
		switch {
		case name == "Vyatta Component":
			err = parseComponent(section, config)
			if err != nil {
				return nil, err
			}

		case strings.HasPrefix(section.Name(), busPrefix):
			/*
			   [Model net.vyatta.test.example]
			   Modules=example-v1,example-interfaces-v1
			   ModelSet=vyatta-v1,vyatta-v2
			   ImportsRequiredForCheck=foo-v1,bar-v2

			   [Model org.ietf.test.example]
			   Modules=ietf-example
			   ModelSet=ietf-v1
			*/
			model := &Model{
				Name:      section.Name()[len(busPrefix):],
				ExecName:  config.ExecName,
				Modules:   section.Key("Modules").Strings(","),
				ModelSets: section.Key("ModelSets").Strings(","),
				ImportsForCheck: section.Key(
					"ImportsRequiredForCheck").Strings(","),
			}
			config.ModelByName[model.Name] = model
			for _, m := range model.ModelSets {
				config.ModelByModelSet[m] = model
			}
		}
	}

	return config, nil
}

type MissingFieldError error

func missingField(section, field string) MissingFieldError {
	return fmt.Errorf("Missing %s field from %s section", field, section)
}

func checkTrueOrFalse(value string) (bool, error) {
	if strings.ToLower(value) == "true" {
		return true, nil
	}
	if strings.ToLower(value) == "false" {
		return false, nil
	}
	return false, fmt.Errorf("Value '%s' must be 'true' or 'false'", value)
}

// parseCSVs parses and validates comma-separated-value fields
// (eg Before, After and ConfigFile)
//
// Allows for trailing commas, and unwanted whitespace adjacent to commas.
// Spaces between words are not allowed as no valid names can have spaces.
func parseCSVs(csvs string) ([]string, error) {
	splitCSVs := strings.Split(csvs, ",")
	var retCsvs []string

	for _, csv := range splitCSVs {
		csv = strings.TrimSpace(csv)
		if csv == "" {
			continue
		}
		if strings.Contains(csv, " ") {
			return nil, fmt.Errorf("Entries may not contain spaces: '%s'",
				csv)
		}
		retCsvs = append(retCsvs, csv)
	}
	return retCsvs, nil
}

const dotService = ".service"

func addDotService(svcList []string) {
	for index, svc := range svcList {
		if strings.HasSuffix(svc, dotService) {
			continue
		}
		svcList[index] += dotService
	}
}

func parseComponent(section *ini.Section, config *ServiceConfig) error {

	for _, field := range section.KeyStrings() {

		value := section.Key(field).String()
		switch field {
		case "Description":
			config.Description = value
		case "Name":
			config.Name = value
			if strings.HasSuffix(value, dotService) {
				return fmt.Errorf("Component Name must not include '.service'")
			}
		case "ExecName":
			config.ExecName = value
		case "ConfigFile":
			cfgFiles, err := parseCSVs(value)
			if err != nil {
				return fmt.Errorf(
					"Unable to parse 'ConfigFile': '%s'\nError: %s",
					value, err.Error())
			}
			config.ConfigFiles = cfgFiles
		case "Before":
			before, err := parseCSVs(value)
			addDotService(before)
			if err != nil {
				return fmt.Errorf("Unable to parse 'Before': '%s'\nError: %s",
					value, err.Error())
			}
			config.Before = before
		case "After":
			after, err := parseCSVs(value)
			addDotService(after)
			if err != nil {
				return fmt.Errorf("Unable to parse 'After': '%s'\nError: %s",
					value, err.Error())
			}
			config.After = after
		case "StartOnBoot":
			startOnBoot, err := checkTrueOrFalse(value)
			if err != nil {
				return fmt.Errorf("Unable to parse 'StartOnBoot': %s\n",
					err.Error())
			}
			config.StartOnBoot = startOnBoot
		case "Ephemeral":
			ephemeral, err := checkTrueOrFalse(value)
			if err != nil {
				return fmt.Errorf("Unable to parse 'Ephemeral': %s\n",
					err.Error())
			}
			config.Ephemeral = ephemeral
		case "DefaultComponent":
			isDefaultComp, err := checkTrueOrFalse(value)
			if err != nil {
				return fmt.Errorf("Unable to parse 'DefaultComponent': %s\n",
					err.Error())
			}
			config.DefaultComp = isDefaultComp
		}
	}

	// Check mandatory fields
	if config.Description == "" {
		return missingField(section.Name(), "Description")
	}
	if config.Name == "" {
		return missingField(section.Name(), "Name")
	}
	if config.ExecName == "" && !config.Ephemeral {
		return missingField(section.Name(), "ExecName")
	}

	return nil
}
