// Copyright (c) 2018-2019, AT&T Intellectual Property.  All rights reserved.
//
// Copyright (c) 2016-2017 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package conf

import (
	"bytes"
	"strings"

	"github.com/danos/vci/services"
	"github.com/go-ini/ini"
)

const ephemeraService = "ephemerad.service"

func (comp *ServiceConfig) GenerateSystemdService() []byte {

	cfg := ini.Empty()

	cfg_unit, _ := cfg.NewSection("Unit")
	cfg_unit.NewKey("Description", comp.Description)
	if len(comp.Before) > 0 {
		cfg_unit.NewKey("Before", strings.Join(comp.Before, " "))
	}
	comp.After = append(comp.After, "vyatta-vci-bus.service")
	if comp.Ephemeral {
		comp.After = append(comp.After, ephemeraService)
		cfg_unit.NewKey("PartOf", ephemeraService)
	}
	cfg_unit.NewKey("After", strings.Join(comp.After, " "))
	cfg_unit.NewKey("BindsTo", "vyatta-vci-bus.service")

	cfg_service, _ := cfg.NewSection("Service")
	cfg_service.NewKey("Type", "notify")
	cfg_service.NewKey("Restart", "on-failure")
	cfg_service.NewKey("ExecStart", comp.ExecName)
	if comp.Ephemeral {
		if comp.ExecName == "" {
			cfg_service.NewKey("ExecStart",
				"/lib/vci/ephemera/bin/activate -component "+comp.Name)
		}
		cfg_service.NewKey("ExecStop", "/lib/vci/ephemera/bin/deactivate -component "+comp.Name)
		cfg_service.NewKey("RemainAfterExit", "true")
	}
	cfg_install, _ := cfg.NewSection("Install")
	if comp.StartOnBoot {
		wantedBy := []string{services.MultiUserTarget}
		if comp.Ephemeral {
			wantedBy = append(wantedBy, ephemeraService)
		}
		cfg_install.NewKey("WantedBy", strings.Join(wantedBy, " "))
	}

	aliases := []string{comp.Name + ".service"}

	for mod_name, _ := range comp.ModelByName {
		aliases = append(aliases, mod_name+".service")
	}
	cfg_install.NewKey("Alias", strings.Join(aliases, " "))

	output := bytes.NewBuffer(nil)
	cfg.WriteTo(output)
	return output.Bytes()
}
