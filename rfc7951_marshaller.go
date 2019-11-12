// Copyright (c) 2017-2019, AT&T Intellectual Property. All rights reserved.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package vci

import (
	"bytes"
	"github.com/danos/encoding/rfc7951"
)

type rfc7951Marshaller struct{}

func (m *rfc7951Marshaller) Marshal(object interface{}) (string, error) {
	buf, err := rfc7951.Marshal(object)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func (m *rfc7951Marshaller) Unmarshal(data string, object interface{}) error {
	return rfc7951.Unmarshal([]byte(data), object)
}

func (m *rfc7951Marshaller) IsEmptyObject(data string) bool {
	var buf bytes.Buffer
	rfc7951.Compact(&buf, []byte(data))
	return buf.String() == "{}"
}

func newRFC7951Marshaller() *rfc7951Marshaller {
	return &rfc7951Marshaller{}
}
