# VCI Debian Helper

## Overview

The VCI Debian Helper consists of 2 distinct parts:

  * Build-time helper (see dh-vci repo)
  * Package install / remove helper
    * deb-vci-helper
	* conf (DBUS and systemd config file generator)

The build-time helper sets up the package(s) being built so that their
'dotComponent' files are installed to the correct location, and the
package install/remove utility (deb-vci-helper) is scheduled to run
'postinst' or 'prerm' to convert the 'dotComponent' file(s) into the
relevant resource files described below.

## deb-vci-helper

deb-vci-helper is used to install or remove configuration files for a
VCI component, based on the component's 'dotComponent' file.  Naming is
very important here:

  * <dotComponentFilename> - prefix of dotComponentFilename (upto '.')
  * <componentName> - name of component as defined in <dotComponentFilename>

The following files are currently created:

  * Per Component
    * systemd
      * <dotComponentFilename>.service
  	  * <componentName> symlink to above file
    * DBUS
      * service
  	  * conf
  * Per Model in each 'dotComponent' file
    * systemd
	  * symlink to where?
	* DBUS
	  * service
	  * conf

