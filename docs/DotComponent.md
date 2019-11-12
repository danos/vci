# Dot Component Files

The Dot Component file describes how your service integrates into BVNOS, what
message bus names are used, where the configuration is stored, etc.

The Dot Component file uses the INI file format, with a main section called
"Vyatta Component". This defines the service's main parameters. This is then
followed by one or more "Model" sections, which define the interfaces that the
service is available on.

One model may be part of multiple modelsets, but 2 models within one Dot
Component file may not be part of the same modelset.  This means one model
must present an identical API to each modelset.

## Note on naming

  * component file name is arbitrary, though it should be closely related
    to the <Name> field in the file.
  * <Name> in file MUST match what is passed to vci.NewComponent(), and
    the <Model> that supports vyatta-v1 MUST be called <Name>.v1.

## Section: Vyatta Component

#### Name
The main namespace of the service.  This is the component name that will be
registered via vci.NewComponent(), and will (with '.service' added) become
the registered service name on systemd.

#### Description
A brief description of the service

#### ExecName
The executable that implements the service

#### ConfigFile
The configuration file(s) where the active state is stored.  You may specify
multiple files, in which case these should be comma-separated.

NB: on initial install, and on rollback, any configuration file listed here
    WILL BE REMOVED.  This is because in these scenarios, the central
	configuration system owns the system config, and will write out the
	new config to each component following the reboot.  At all other times,
	the component owns the configuration, and on a standard reboot, any
	component with active configuration will boot independently of the central
	configuration system (path-based activation, not yet implemented).

NB: For Acton release only, '.json' configuration files will actually be
    replaced with an empty JSON file ('{}') rather than being completely
	removed.

#### Before
The service that this file represents will start BEFORE services listed
here.  Format for service names is as noted above.  Multiple entries should
be comma-separated.

#### After
The service that this file represents will start AFTER services listed
here.  Format for service names is as noted above.  Multiple entries should
be comma-separated.

#### StartOnBoot
If set to true, this component will always be started on boot, regardless
of whether it has any non-default configuration (ie anything that is
shown in the overall configuration).  Otherwise the component will be
started only when configured.

## Section: Model \<name\>

#### \<name\>
The namespace that the model presents. At the moment, only one model
is supported, and it must provide the "vyatta-v1" model set.

#### Modules
Comma-separated list of the YANG modules and submodules that this model
provides.  Note that submodules need to be explicitly listed, as it is
possible for submodules to belong to a different component to the parent
module, or for one to belong to provisiond and one to another component.

#### ModelSets
Comma-separated list of the presented model sets that this model supports.

## Example

```
[Vyatta Component]
Name=net.vyatta.vci.example.spaceteam
Description=Super Example Project
ExecName=/opt/vyatta/sbin/spaceteam
ConfigFile=/etc/vyatta/spaceteam.conf

[Model net.vyatta.vci.example.spaceteam.v1]
Modules=spaceteam-v1
ModelSets=vyatta-v1
```
