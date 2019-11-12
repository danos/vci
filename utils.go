// Copyright (c) 2017-2019, AT&T Intellectual Property. All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package vci

import (
	"bytes"
	"reflect"
	"sync"
	"sync/atomic"
	"unicode"
	"unicode/utf8"
)

var (
	reflectStringType    = reflect.TypeOf("")
	reflectByteSliceType = reflect.TypeOf([]byte(nil))
	reflectErrorType     = reflect.TypeOf((*error)(nil)).Elem()
)

type multiWriterValue struct {
	atom    atomic.Value
	writelk sync.Mutex
}

func newMultiWriterValue(initialData interface{}) *multiWriterValue {
	out := &multiWriterValue{}
	out.atom.Store(initialData)
	return out
}

func (v *multiWriterValue) Load() interface{} {
	return v.atom.Load()
}

func (v *multiWriterValue) Update(fn func(interface{}) interface{}) {
	v.writelk.Lock()
	v.atom.Store(fn(v.atom.Load()))
	v.writelk.Unlock()
}

// genYangName maps Go names to YANG names. CamelCase to camel-case.
func genYangName(name string) string {
	end := utf8.RuneCountInString(name)
	if end == 0 {
		return name
	}
	end-- // end is the index of the last rune of name

	var prev rune
	var buf []byte
	b := bytes.NewBuffer(buf)
	for i, r := range name {
		// If we transition from a letter to a number (but not
		// at the end of the name) or we transition from a
		// lowercase letter to an uppercase rune, inject a
		// hyphen
		if (unicode.IsNumber(r) && unicode.IsLetter(prev) && i != end) ||
			(unicode.IsUpper(r) && unicode.IsLetter(prev) && !unicode.IsUpper(prev)) {
			b.WriteRune('-')
		}
		b.WriteRune(unicode.ToLower(r))
		prev = r
	}
	return b.String()
}

// newGoObject creates an object based on the input type. There is some
// subtle behavior that needs to be accounted for when creating some
// types for use in unmarshalling.
//  (1) When creating a new object of a pointer type one creates a
//      new object of the type it points to.
//  (2) For everything else a new type is allocated and a pointer
//      is returned to it, for some types this might seem wrong
//      ('int's for instance) but the wireformat assumes everything to
//      be in a container and will fail to unmarshal if it isn't so
//      this case does not matter.
func newGoObject(ty reflect.Type) interface{} {
	switch ty.Kind() {
	case reflect.Ptr:
		return reflect.New(ty.Elem()).Interface()
	default:
		return reflect.New(ty).Interface()
	}
}

func decodeValue(typ reflect.Type, encodedData string) (interface{}, error) {
	switch typ {
	case reflectByteSliceType:
		return []byte(encodedData), nil
	case reflectStringType:
		return encodedData, nil
	default:
		marshaller := defaultMarshaller()
		if marshaller.IsEmptyObject(encodedData) && typ.Kind() == reflect.Ptr {
			// Pass nil when the incoming object is empty.
			// This is verbose using reflect.
			value := reflect.New(typ)            // value = new(*T)  //value's type is **T
			value.Elem().Set(reflect.Zero(typ))  // *value = *T(nil)
			return value.Elem().Interface(), nil // return *value, nil
		}
		value := newGoObject(typ)
		err := marshaller.Unmarshal(encodedData, value)
		if err != nil {
			return nil, err
		}
		val := reflect.ValueOf(value)
		valType := val.Type()
		if valType.Kind() == reflect.Ptr && typ == valType.Elem() {
			return val.Elem().Interface(), nil
		} else {
			return val.Interface(), nil
		}
	}
}
