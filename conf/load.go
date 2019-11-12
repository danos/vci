// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2016 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package conf

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

func getConfigFilenames(dir string) ([]string, error) {

	fi, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsDir() {
		return nil, errors.New("Not a directory")
	}
	d, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	names, err := d.Readdirnames(0)
	if err != nil {
		return nil, err
	}
	fnames := make([]string, 0)
	for _, name := range names {
		if !strings.HasSuffix(name, ".component") {
			continue
		}
		fname := dir + "/" + name
		fnames = append(fnames, fname)
	}

	return fnames, nil
}

func loadComponentFile(file string) (*ServiceConfig, error) {

	contents, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading component file %s:\n  %s\n\n",
			file, err.Error())
	}

	comp, err := ParseConfiguration(contents)
	if err != nil {
		return nil, err
	}

	return comp, nil
}

func LoadComponentConfigDir(dir string) ([]*ServiceConfig, error) {

	fnames, err := getConfigFilenames(dir)
	if err != nil {
		return nil, err
	}

	components := make([]*ServiceConfig, 0, len(fnames))
	for _, f := range fnames {
		comp, err := loadComponentFile(f)
		if err != nil {
			return nil, err
		}
		components = append(components, comp)
	}

	return components, nil
}
