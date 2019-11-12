// Copyright (c) 2018-2019, AT&T Intellectual Property. All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0
//

package services

import (
	"fmt"
	"os"
	"strings"

	systemd "github.com/coreos/go-systemd/dbus"
)

// Manager - used to manage a connection to systemd
// 'available' is determined when NewManager() is called and not retested
// for the assumed short (one command, or one command per component in a loop)
// lifetime of the object.  Close() should be called when finished.
type Manager struct {
	conn      *systemd.Conn
	available bool
}

const (
	serviceSuffix     = ".service"
	serviceFileDir    = "/lib/systemd/system/"
	systemdSymlinkDir = "/etc/systemd/system/"
	MultiUserTarget   = "multi-user.target"
)

func NewManager() *Manager {
	mgr := &Manager{}
	mgr.checkAvailability()
	return mgr
}

func (mgr *Manager) isAvailable() bool { return mgr.available }

// checkAvailability() - check if systemd is running.
//
// We may be called during the 'build-iso' stage of the build, when
// packages are installed in bulk, and systemd is not running. In this
// case we want to silently ignore requests to carry out DBUS operations.
// The lifetime of a connection is short enough that we do not expect
// systemd to spring to life part-way through.
//
func (mgr *Manager) checkAvailability() {

	mgr.available = true

	if _, err := os.Stat("/run/systemd/system"); err != nil {
		if os.IsNotExist(err) {
			mgr.available = false
		}
		// Any other error indicates that the path exists, though we may not
		// have permission to look at it!  Carry on ...
	}
}

// connect - attempt to connect to systemd
//
// Return error if systemd is available and we cannot connect.  It is not
// an error if systemd is not available - we might be installing for example.
func (mgr *Manager) connect() error {
	if mgr.conn != nil || !mgr.isAvailable() {
		return nil
	}

	c, e := systemd.NewSystemdConnection()
	if e != nil {
		return e
	}
	mgr.conn = c
	return nil
}

// Close - should be called when finished with a Manager object
func (mgr *Manager) Close() {
	if mgr.conn != nil {
		mgr.conn.Close()
	}
}

// Rather than insist users append '.service', do it as required.
func (mgr *Manager) svcName(name string) string {
	if !strings.HasSuffix(name, serviceSuffix) {
		return name + serviceSuffix
	}
	return name
}

func (mgr *Manager) Start(name string) error {
	if err := mgr.connect(); err != nil {
		return err
	}
	if !mgr.isAvailable() {
		return nil
	}
	_, err := mgr.conn.StartUnit(mgr.svcName(name), "replace", nil)
	return err
}

func (mgr *Manager) Reload(name string) error {
	if err := mgr.connect(); err != nil {
		return err
	}
	if !mgr.isAvailable() {
		return nil
	}
	_, err := mgr.conn.ReloadUnit(mgr.svcName(name), "replace", nil)
	return err
}

func (mgr *Manager) ReloadOrRestart(name string) error {
	if err := mgr.connect(); err != nil {
		return err
	}
	if !mgr.isAvailable() {
		return nil
	}
	_, err := mgr.conn.ReloadOrRestartUnit(mgr.svcName(name), "replace", nil)
	return err
}

func (mgr *Manager) Restart(name string) error {
	if err := mgr.connect(); err != nil {
		return err
	}
	if !mgr.isAvailable() {
		return nil
	}
	_, err := mgr.conn.RestartUnit(mgr.svcName(name), "replace", nil)
	return err
}

// Looks for changed service files and reloads them.
func (mgr *Manager) ReloadServices() error {
	if err := mgr.connect(); err != nil {
		return err
	}
	if !mgr.isAvailable() {
		return nil
	}
	err := mgr.conn.Reload()
	return err
}

func (mgr *Manager) Stop(name string) error {
	if err := mgr.connect(); err != nil {
		return err
	}
	if !mgr.isAvailable() {
		return nil
	}
	_, err := mgr.conn.StopUnit(mgr.svcName(name), "replace", nil)
	return err
}

// FileName - service file name (service definition)
func FileName(service string) string {
	return serviceFileDir + service + ".service"
}

func (mgr *Manager) svcSymlinkPath(name string) string {
	return systemdSymlinkDir + MultiUserTarget + ".wants/" + name + ".service"
}

// Enable - Enable service so it will start on boot.
// This will be done by systemd, if running; otherwise the required
// symlink will be created manually.  Note that it is assumed that
// it is the multi-user target we are interested in as this is what
// our component files specify (see vci/conf).
func (mgr *Manager) Enable(name string) error {
	if err := mgr.connect(); err != nil {
		return err
	}
	if !mgr.isAvailable() {
		if err := os.Symlink(
			FileName(name), mgr.svcSymlinkPath(name)); err != nil {
			return fmt.Errorf("Cannot enable '%s': %s\n", name, err.Error())
		}
		return nil
	}
	_, _, err := mgr.conn.EnableUnitFiles(
		[]string{mgr.svcName(name)},
		false, /* persistent */
		true /* force symlink replacement */)
	return err
}

// See comments on Enable() which apply here too.
func (mgr *Manager) Disable(name string) error {
	if err := mgr.connect(); err != nil {
		return err
	}
	if !mgr.isAvailable() {
		if err := os.Remove(mgr.svcSymlinkPath(name)); err != nil {
			return fmt.Errorf("Cannot disable '%s': %s\n", name, err.Error())
		}
	}
	_, err := mgr.conn.DisableUnitFiles(
		[]string{mgr.svcName(name)},
		false /* persistent */)
	return err
}

func (mgr *Manager) IsActive(name string) (bool, error) {
	if err := mgr.connect(); err != nil {
		return false, err
	}
	if !mgr.isAvailable() {
		return false, nil
	}
	unitsStatus, err := mgr.conn.ListUnitsByNames([]string{mgr.svcName(name)})
	if err != nil {
		return false, err
	}
	if len(unitsStatus) != 1 {
		return false, fmt.Errorf("Unable to determine status of %s", name)
	}
	if unitsStatus[0].ActiveState == "active" {
		return true, nil
	}
	return false, nil
}
