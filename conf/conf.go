// Copyright (c) 2018-2019, AT&T Intellectual Property. All rights reseved.
//
// Copyright (c) 2016-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package conf

type Model struct {
	Name      string
	ExecName  string
	ModelSets []string
	Modules   []string
}

type ServiceConfig struct {
	Description     string
	Name            string
	ExecName        string
	ConfigFiles     []string
	Before          []string // This service starts BEFORE services listed here
	After           []string // This service starts AFTER services listed here
	StartOnBoot     bool
	Ephemeral       bool
	DefaultComp     bool
	ModelByName     map[string]*Model
	ModelByModelSet map[string]*Model
}
