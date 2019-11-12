Integration Library - General
=============================
The integration libraries provide a set of easy to use abstractions that make
it easy to create a new Vyatta Component which integrates with the BVNOS.

The exact API details vary slightly for each specific implementation language,
but the general concepts are the same across each language specific library.


Overview
--------
The structure of the component code should be:

1. Create a new "component" object
1. Register each of its models
1. Run the component


Create a Component
------------------
```
comp = vci.NewComponent("net.vyatta.eng.service.example")
```

This provides a handle for your Component object, which is used to run your
service through a set of registered callbacks. At this stage, the main name of
your service is provided, but no models are registered and the service is not
running.


Register the Models
-------------------
```
comp.RegisterModel("net.vyatta.eng.service.example.v1",
                   ConfigObject,
                   StateObject)
```

This registers a set of objects as the handlers for any requests directed at
the named service. This will receive requests from the YANG modules associated
with this name in the [DotComponent file](DotComponent.md).

The ConfigObject must implement the Config Object Interface [See Config Object
Interface].
The StateObject must implement the State Object Interface [See State Object
Interface].

This step needs to be repeated for each Model that this component provides.


Run the Component
-----------------
```
comp.Run()
```

After all the objects and names have registered then the component should be
activated using the Run() method. This will bring up the service, registering
items in the appropriate order so that any queued requests are handled
properly.

T structures
------------
A "T" in the below definitons are arbitrary structures.  T's must be
JSON encodable using the go "encoding/json" library. If T is a []byte
or a string, it will be passed to and from the feature logic untouched
by the infrastructure. This is an advanced use of the library and
should only be used as a last resort.

Config Object Interface
-----------------------
Implements the configuration tree as specified by the YANG modules for this
Model.

### error Check(T new_config)
Validates the candidate configuration, and returns an error for any
discovered issues. The provided configuration should be valid
according to the YANG specification and only checks that can't be
modelled in the YANG need to be implemented here.

These checks MUST NOT use any state to perform the checks. It is
expected that the same configuration will always be valid. Inter-model
dependencies MUST be modelled in the YANG.

Note that Check() will only be called if the component has active
configuration (candidate or committed) or is already running.

### error Set(T new_config)
Sets a new running configuration. The same validation as for "Check"
may be carried out, however a valid configuration MUST be
accepted. The provisioning step MAY be done asynchronously, in which
case Notifications [See Notifications] SHOULD be used to tell other
components that the provisioning state has changed.


### T Get(string path)
Returns a structure representing current configuration for the
requested path.


State Object Interface
----------------------
Implements the state tree as specified by the YANG modules for this
Model.

### T Get(string path)
Returns a structure representing current state for the requested path.

T's must be JSON encodable using the go "encoding/json" library.


RPC Object Interface
--------------------
Implements a set of arbitrary RPC calls. An RPC method must be of the
following form.

### (T, error) RPC(T input_data)
Takes a structure representing the RPC input data, returns a structure
representing the RPC output data.

The input data will be validated and expanded based on the data model
before being handed to the implementing method.


Notificiations
--------------
Not currently implemented.
