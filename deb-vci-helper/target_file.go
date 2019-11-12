// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2016 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"io/ioutil"
	"os"
)

// Abstraction for different file types with actions for install and remove
type targetFile interface {
	Create()
	Delete()
}

// A concrete file with name and contents
type configFile struct {
	name string
	data func() []byte // Function to create contents when needed
}

func (f *configFile) Create() {
	ioutil.WriteFile(f.name, f.data(), 0644)
}

func (f *configFile) Delete() {
	os.Remove(f.name)
}

// A symlink with name and source file
type symlink struct {
	name   string
	source string
}

func (f *symlink) Create() {
	os.Symlink(f.source, f.name)
}

func (f *symlink) Delete() {
	os.Remove(f.name)
}
