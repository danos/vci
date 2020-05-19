// Copyright (c) 2020, AT&T Intellectual Property. All rights reserved.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/danos/vci"
)

func exitOnError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usage() {
	const usageFmt = `usage %s module-name notification-name [rfc7951-encoded-body]`
	fmt.Fprintf(os.Stderr, usageFmt+"\n", os.Args[0])
	os.Exit(1)
}

func emitNotification(module, name, data string) error {
	client, err := vci.Dial()
	if err != nil {
		return err
	}
	defer client.Close()
	return client.Emit(module, name, data)
}

func main() {
	if len(os.Args) < 3 || len(os.Args) > 4 {
		usage()
	}

	var data string
	switch len(os.Args) {
	case 3:
		b, err := ioutil.ReadAll(os.Stdin)
		exitOnError(err)
		data = strings.TrimSpace(string(b))
	default:
		data = os.Args[3]
	}
	err := emitNotification(os.Args[1], os.Args[2], data)
	exitOnError(err)
}
