# 'Dot Component' file format

## Introduction

'.component' files are used to provide information about a VCI
component, including what models it provides, and which model sets
are supported.

## Format

An example component file is shown:

```ini
  [Vyatta Component]
  Name=net.vyatta.vci.config.exampled
  Description=Example component
  ExecName=/usr/sbin/exampled
  ConfigFile=/etc/vyatta/exampled.conf
  Before=net.vyatta.vci.config.componentThatRunsAfterMe
  After=net.vyatta.vci.config.componentThatRunsBeforeMe

  [Model net.vyatta.vci.config.exampled.v1]
  Modules=example-main-v1,example-extra-v1
  ModelSets=vyatta-v1,other-v1
  ImportsRequiredForCheck=foo-v1,bar-v2

  [Model net.vyatta.vci.config.exampled.v2]
  Modules=example-main-v1,example-extra-v2,example-new-v1
  ModelSets=vyatta-v2
```

This is for the 'exampled' component, which provides 2 different models.

## 'Vyatta Component' fields

Each '.component' file should contain exactly one of these sections.  It
contains basic information about the component.

### Name

Name of the component that will be used to register with the VCI
infrastructure.  We follow the [DBUS bus name spec](https://dbus.freedesktop.org/doc/dbus-specification.html#message-protocol-names-bus), though note that this does not tie
us to using DBUS.  Name must match the component name used in the call to
vci.NewComponent().

### Description

Free-text description

### ExecName

Name of the executable to be run when this component

### ConfigFile

Fully-qualified path for the component's config file.  This should
typically live in /etc/vyatta/.

### Before

Other components that this component should be started before, ie any
component listed here is started after this component.  Names here should
match the names in the relevant component's 'Name' field, comma-separated.

### After

Opposite of 'Before' - a list of components that must be started before
this component.  Names here should match the names in the 'Name' field,
comma-separated.

### DefaultComponent

This is used to provide support for one component to own all YANG modules
that are not explicitly claimed by any VCI component.  Obviously only one
component may be the default.  This is primarily to provide a way to assign
ownership of legacy Vyatta-v1 modelset modules to provisiond.

The default component cannot list any modules explicitly.

## 'Model' fields

Each component may provide one or more models.  These each represent a view
(or interface) of the component that consists of a set of YANG modules.

Different models may contain the same YANG modules.

A model may belong to one or more Model sets, if it wishes to present the
same interface for each model set.  However, it may only provide a single
model per model set.

### model-name (part of Model header)

Unique name of the model.  Must match name used in call to
comp.RegisterModel()

### Modules

Comma-separated list of YANG modules provided by this model.

### ModelSets

Comma-separated list of model sets supported by this model.

### ImportsRequiredForCheck

Optional field used when a component needs candidate configuration from other
components to be able to carry out the check() function.  Content is a comma-
separated list of YANG modules required.

