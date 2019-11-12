// Copyright (c) 2018-2019, AT&T Intellectual Property. All rights reserved.
//
// Copyright (c) 2016 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"github.com/danos/vci/conf"
	"github.com/danos/vci/services"
	"io/ioutil"
	"os"
)

func componentFileName(name string) string {
	return "/lib/vci/components/" + name + ".component"
}

func dbusServiceFileName(name string) string {
	return "/usr/share/dbus-1/system-services/" + name + ".service"
}

func dbusConfigFileName(name string) string {
	return "/etc/vci/bus.d/" + name + ".conf"
}

func usage() {
	fmt.Printf(
		`Usage:
  %s <action> <component>
      <action>            "install" or "remove"
      <component config>  component to install or remove
`, os.Args[0])
	os.Exit(1)
}

func main() {

	if len(os.Args) != 3 {
		usage()
	}

	action := os.Args[1]
	component := os.Args[2]

	if err := processRequest(action, component); err != nil {
		// In the 'remove' case, we can't return an error or uninstall will
		// fail, thus blocking reinstallation of a working version.  We also
		// suppress the error as we would only get it if we were uninstalling
		// as part of a failed install, in which case we'd have already
		// printed the same error for the install.
		if action != "remove" {
			fmt.Fprintf(os.Stderr, err.Error())
			os.Exit(1)
		}
		os.Exit(0)
	}
}

func processRequest(action, component string) (err error) {

	component_file := componentFileName(component)
	dat, err := ioutil.ReadFile(component_file)
	if err != nil {
		return fmt.Errorf("Error reading component file %s:\n  %s\n\n",
			component_file, err.Error())
	}

	compCfg, err := conf.ParseConfiguration(dat)
	if err != nil {
		return fmt.Errorf("Unable to parse %s:\n\t%s\n",
			component_file, err.Error())
	}
	files := getFiles(component, compCfg)

	switch action {
	case "install":
		install(component, files)

		svcMgr := services.NewManager()
		defer svcMgr.Close()

		if compCfg.StartOnBoot {
			// Ensure component will get started on a (re)boot
			if err := svcMgr.Enable(component); err != nil {
				return fmt.Errorf("Unable to enable %s: %s\n",
					component, err.Error())
			}
			// Start if not running, restart if already running.
			if err := svcMgr.ReloadOrRestart(component); err != nil {
				return fmt.Errorf("Unable to reload/(re)start %s: %s\n",
					component, err.Error())
			}
		} else {
			// Only restart if currently running.
			active, err := svcMgr.IsActive(component)
			if active {
				if err := svcMgr.ReloadOrRestart(component); err != nil {
					return fmt.Errorf("Unable to reload/restart %s: %s\n",
						component, err.Error())
				}
			} else if err != nil {
				return fmt.Errorf("Unable to determine state of %s: %s\n",
					component, err.Error())
			}
		}
	case "remove":
		remove(component, files)

		svcMgr := services.NewManager()
		defer svcMgr.Close()

		// Make sure we clean up the symlink first so systemd won't try to
		// start it next time.
		if err := svcMgr.Disable(component); err != nil {
			return fmt.Errorf("Unable to disable %s: %s\n",
				component, err.Error())
		}

		active, err := svcMgr.IsActive(component)
		if active {
			if err := svcMgr.Stop(component); err != nil {
				return fmt.Errorf("Unable to stop %s: %s\n",
					component, err.Error())
			}
		} else if err != nil {
			return fmt.Errorf("Unable to determine state of %s: %s\n",
				component, err.Error())
		}
	default:
		usage()
	}

	return nil
}

func systemdConfigFile(component string, compCfg *conf.ServiceConfig) targetFile {
	return &configFile{
		services.FileName(component),
		func() []byte { return compCfg.GenerateSystemdService() },
	}
}

func dbusFiles(service conf.DbusService) []targetFile {

	return []targetFile{
		&configFile{
			dbusServiceFileName(service.FilePrefix()),
			func() []byte { return service.GenerateDbusService() },
		},
		&configFile{
			dbusConfigFileName(service.FilePrefix()),
			func() []byte { return service.GenerateDbusConfig() },
		},
	}
}

func systemdAlias(name, alias string) targetFile {
	return &symlink{
		services.FileName(alias),
		services.FileName(name),
	}

}
func getFiles(component string, compCfg *conf.ServiceConfig) []targetFile {
	cfg_files := []targetFile{}

	// Add main component SystemD service
	cfg_files = append(cfg_files,
		systemdConfigFile(component, compCfg))

	//  Add main component Systemd Alias
	cfg_files = append(cfg_files,
		systemdAlias(component, compCfg.Name))

	// Add main component D-Bus Service and Config files
	cfg_files = append(cfg_files,
		dbusFiles(compCfg)...)

	// For each module add the Dbus files and Systemd Alias
	for _, mod := range compCfg.ModelByName {
		cfg_files = append(cfg_files, systemdAlias(component, mod.Name))
		cfg_files = append(cfg_files, dbusFiles(mod)...)
	}
	return cfg_files
}

func install(component string, files []targetFile) {
	for _, file := range files {
		file.Create()
	}

	// Make sure this is done prior to (re)start so we are using updated
	// files.
	dbusAndDaemonReload()

	svcMgr := services.NewManager()
	defer svcMgr.Close()
	active, err := svcMgr.IsActive(component)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"Unable to determine service state for %s: %s\n",
			component, err.Error())
		os.Exit(1)
	}
	if !active {
		return
	}

	if err := svcMgr.ReloadOrRestart(component); err != nil {
		fmt.Fprintf(os.Stderr,
			"Unable to reload / (re)start %s: %s\n",
			component, err.Error())
		os.Exit(1)
	}
}

func remove(component string, files []targetFile) {
	defer dbusAndDaemonReload()

	for _, file := range files {
		file.Delete()
	}

	// Stop before DBUS and daemon reload.
	svcMgr := services.NewManager()
	defer svcMgr.Close()
	fmt.Fprintf(os.Stdout, "Checking status of %s before removal.\n",
		component)
	active, err := svcMgr.IsActive(component)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"Unable to determine service state for %s: %s\n",
			component, err.Error())
		os.Exit(1)
	}
	if !active {
		return
	}

	fmt.Fprintf(os.Stdout, "About to stop %s.\n",
		component)
	if err := svcMgr.Stop(component); err != nil {
		fmt.Fprintf(os.Stderr,
			"Unable to stop %s: %s\n",
			component, err.Error())
		os.Exit(1)
	}
}

func dbusAndDaemonReload() {
	svcMgr := services.NewManager()
	defer svcMgr.Close()

	if err := svcMgr.ReloadServices(); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to run daemon-reload: %s\n",
			err.Error())
		os.Exit(1)
	}

	if err := svcMgr.Reload("vyatta-vci-bus"); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to restart DBUS service: %s\n",
			err.Error())
		os.Exit(1)
	}
}
