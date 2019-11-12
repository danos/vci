// Copyright (c) 2018-2019, AT&T Intellectual Property.
// All rights reserved.
// Copyright (c) 2016 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package conf

import (
	"bytes"
	"fmt"
	"github.com/go-ini/ini"
)

type DbusService interface {
	FilePrefix() string
	GenerateDbusService() []byte
	GenerateDbusConfig() []byte
}

func generateDbusServiceFile(name, execName string) []byte {
	cfg := ini.Empty()

	cfg_dbus, _ := cfg.NewSection("D-BUS Service")
	cfg_dbus.NewKey("Notify", "true")
	cfg_dbus.NewKey("Name", name)
	cfg_dbus.NewKey("Exec", "/bin/systemctl start "+name)
	cfg_dbus.NewKey("User", "root")
	cfg_dbus.NewKey("SystemdService", name+".service")

	output := bytes.NewBuffer(nil)
	cfg.WriteTo(output)

	return output.Bytes()
}

func (comp *ServiceConfig) FilePrefix() string {
	return comp.Name
}

func (comp *ServiceConfig) GenerateDbusService() []byte {
	return generateDbusServiceFile(comp.Name, comp.ExecName)
}

func (comp *ServiceConfig) GenerateDbusConfig() []byte {
	template := `<!DOCTYPE busconfig PUBLIC
 "-//freedesktop//DTD D-BUS Bus Configuration 1.0//EN"
 "http://www.freedesktop.org/standards/dbus/1.0/busconfig.dtd">
<busconfig>
	<policy user="root">
		<allow own="%s"/>
		<allow send_destination="*"/>
	</policy>
</busconfig>
`

	config := fmt.Sprintf(template, comp.Name)
	return []byte(config)
}

func (mod *Model) FilePrefix() string {
	return mod.Name
}

func (mod *Model) GenerateDbusService() []byte {
	return generateDbusServiceFile(mod.Name, mod.ExecName)
}

func (mod *Model) GenerateDbusConfig() []byte {
	template := `<!DOCTYPE busconfig PUBLIC
 "-//freedesktop//DTD D-BUS Bus Configuration 1.0//EN"
 "http://www.freedesktop.org/standards/dbus/1.0/busconfig.dtd">
<busconfig>
	<policy user="root">
		<allow own="%s"/>
		<allow send_destination="*"/>
	</policy>
</busconfig>
`

	config := fmt.Sprintf(template, mod.Name)
	return []byte(config)
}
