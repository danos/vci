# Overview

The Vyatta Component Infrastructure is a set of libraries and helpers that allow
you to easily build and integrate a service into a Brocade Vyatta Network OS
product.

# Dot Component Files

The heart of the process is the "Dot Component" file, which describes your
service and how it will interact with the Vyatta ecosystem. This file names the
service's executable, configuration file and YANG modules.

See [detailed description](DotComponent.md)

# VCI Debian Helper

The VCI Debian Helper takes care of many of the details needed to build a
BVNOS compatible debian package. Specifically, it ensures that the
"Dot Component" file is put in the right place, and the correct scripts
are called during installation and removal, to properly generate and clean
up the system integration.

See [detailed description](VciDebianHelper.md)

# Integration Libraries

The VCI libraries allow you to easily interact with a BVNOS system, in whatever
language you choose (as long as it's Go). Whilst it is possible to directly
interface with the lower level "plumbing", it is strongly recommended to use
one of the libraries to take care of the integration details.

See [detailed description](IntegrationLibrary-General.md)
