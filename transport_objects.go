// Copyright (c) 2017,2019, AT&T Intellectual Property. All rights reserved.
// Copyright (c) 2016 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package vci

import (
	"errors"
	"reflect"
)

type state struct {
	wrapperObject
	err error

	methods map[string]interface{}
}

func newState(object interface{}, client *Client) *state {
	s := &state{
		wrapperObject: wrapperObject{
			client: client,
		},
		methods: make(map[string]interface{}),
	}
	s.generateMethods(object)
	return s
}

func (o *state) generateMethods(object interface{}) {
	const name = "Get"
	var method reflect.Value
	err := o.setMethod(&method, reflect.ValueOf(object), name,
		o.validateGet)
	if err != nil {
		o.err = err
		return
	}
	o.methods[genYangName(name)] = func() (string, error) {
		out := method.Call(
			[]reflect.Value{})[0]
		return o.encodeOutput(out.Interface())
	}
}

func (o *state) Methods() map[string]interface{} {
	return o.methods
}

func (o *state) IsValid() bool {
	return o != nil && o.err == nil
}

func (o *state) Error() string {
	return o.err.Error()
}

func (o *state) Name() string {
	return "state"
}

func (o *state) Type() string {
	return "state"
}

type config struct {
	wrapperObject
	err error

	methods map[string]interface{}
}

func newConfig(object interface{}, client *Client) *config {
	if object == nil {
		return &config{
			err: errors.New("cannot wrap nil object"),
		}
	}
	cfg := &config{
		wrapperObject: wrapperObject{
			client: client,
		},
		methods: make(map[string]interface{}),
	}
	cfg.generateMethods(object)
	return cfg
}

func (o *config) Methods() map[string]interface{} {
	return o.methods
}

func (o *config) IsValid() bool {
	return o != nil && o.err == nil
}

func (o *config) Error() string {
	return o.err.Error()
}

func (o *config) Name() string {
	return "running"
}

func (o *config) Type() string {
	return "config"
}

func (o *config) generateMethods(object interface{}) {
	fns := []func(object interface{}) error{
		o.generateGetMethod,
		o.generateSetMethod,
		o.generateCheckMethod,
	}
	for _, fn := range fns {
		err := fn(object)
		if err != nil {
			o.err = err
			return
		}
	}
}

func (o *config) generateGetMethod(object interface{}) error {
	const name = "Get"
	var method reflect.Value
	objectVal := reflect.ValueOf(object)
	methodFunc := objectVal.MethodByName(name)
	if !methodFunc.IsValid() {
		return nil
	}
	err := o.setMethod(&method, objectVal, name,
		o.validateGet)
	if err != nil {
		return err
	}
	o.methods[genYangName(name)] = func() (string, error) {
		out := method.Call([]reflect.Value{})[0]
		return o.encodeOutput(out.Interface())
	}
	return nil
}

func (o *config) generateSetMethod(object interface{}) error {
	const name = "Set"
	var method reflect.Value
	err := o.setMethod(&method, reflect.ValueOf(object), name,
		o.validateSet)
	if err != nil {
		return err
	}
	o.methods[genYangName(name)] = func(encodedData string) error {
		methodType := method.Type()
		methodInputType := methodType.In(0)

		ins, errs := o.decodeInput(methodInputType, encodedData)
		if errs != nil {
			return errs
		}

		outs := method.Call(ins)

		return o.encodeError(outs[0].Interface())
	}
	return nil
}

func (o *config) generateCheckMethod(object interface{}) error {
	const name = "Check"
	var method reflect.Value
	err := o.setMethod(&method, reflect.ValueOf(object), name,
		o.validateCheck)
	if err != nil {
		return err
	}
	o.methods[genYangName(name)] = func(encodedData string) error {
		methodType := method.Type()
		methodInputType := methodType.In(0)

		ins, errs := o.decodeInput(methodInputType, encodedData)
		if errs != nil {
			return errs
		}

		outs := method.Call(ins)

		return o.encodeError(outs[0].Interface())
	}
	return nil
}

type rpcObject struct {
	wrapperObject
	err  error
	name string

	methods map[string]interface{}
}

func newRPC(moduleName string, object interface{}, client *Client) *rpcObject {
	if object == nil {
		return &rpcObject{
			err: errors.New("cannot wrap nil object"),
		}
	}
	rpc := &rpcObject{
		name: moduleName,
		wrapperObject: wrapperObject{
			client: client,
		},
		methods: make(map[string]interface{}),
	}
	rpc.generateMethods(object)
	return rpc
}

func (o *rpcObject) Methods() map[string]interface{} {
	return o.methods
}

func (o *rpcObject) IsValid() bool {
	return o != nil && o.err == nil
}

func (o *rpcObject) Error() string {
	return o.err.Error()
}

func (o *rpcObject) Name() string {
	return o.name
}

func (o *rpcObject) Type() string {
	return "rpc"
}

func (o *rpcObject) generateMethods(object interface{}) {
	switch v := object.(type) {
	case map[string]interface{}:
		o.generateMethodsFromMap(v)
	default:
		o.generateMethodsFromObject(object)
	}
}

func (o *rpcObject) generateMethodsFromMap(in map[string]interface{}) {
	for name, method := range in {
		methodValue := reflect.ValueOf(method)
		methodType := methodValue.Type()
		if methodValue.Kind() != reflect.Func {
			continue
		}
		if methodType.NumIn() != 1 {
			continue
		}
		if methodType.NumOut() != 2 {
			continue
		}
		if methodType.Out(1) != reflectErrorType {
			continue
		}
		o.methods[name] = o.wrapRPCMethod(o.Name(), name,
			methodValue)
	}
}

func (o *rpcObject) generateMethodsFromObject(object interface{}) {
	value := reflect.ValueOf(object)
	typ := reflect.TypeOf(object)
	for i := 0; i < value.NumMethod(); i++ {
		methodType := value.Method(i).Type()
		methodExpr := typ.Method(i)
		//methods must be of the form func(_) (_, _) we don't
		//care about argument types because they will be
		//encoded/decode by the wrapper function.
		if methodType.NumIn() != 1 {
			o.err = errors.New(
				"All RPCs must have one and only one argument")
			return
		}
		if methodType.NumOut() != 2 {
			o.err = errors.New(
				"All RPCs must have 2 and only 2 returns")
			return
		}
		if methodType.Out(1) != reflectErrorType {
			o.err = errors.New(
				"All RPCs must have the error type as the second return")
		}
		name := genYangName(methodExpr.Name)
		o.methods[name] = o.wrapRPCMethod(o.Name(), name,
			value.Method(i))
	}
}

func (o *rpcObject) wrapRPCMethod(
	moduleName, name string,
	method reflect.Value,
) func(string) (string, error) {
	wrapper := func(encodedData string) (string, error) {
		methodType := method.Type()
		methodInputType := methodType.In(0)
		ok, err := o.validateRPCInput(
			moduleName, name, encodedData, methodInputType)
		if !ok {
			return "", err
		}
		var outs []reflect.Value

		ins, errs := o.decodeInput(methodInputType, encodedData)
		if errs != nil {
			return "", errs
		}

		outs = method.Call(ins)
		val := outs[0]
		errv := outs[1]

		if errv.IsValid() && !errv.IsNil() {
			return "", o.encodeError(errv.Interface())
		}

		return o.encodeOutput(val.Interface())
	}
	return wrapper
}

func (o *rpcObject) validateRPCInput(
	module, name string,
	encodedData string,
	typ reflect.Type,
) (bool, error) {
	switch {
	case typ == reflectStringType:
		return true, nil
	case typ == reflectByteSliceType:
		return true, nil
	case module == "":
		// If there is no module name don't validate the input
		// this cannot be reached from outside, and is used
		// for testing the functionallity of the wrappers.
		// A better way to do this would be to mock out the
		// various calls to YANGd in the testing library.
		// This should be done at a future time.
		return true, nil
	default:
		return o.validateRPCInputByModuleName(
			module, name, encodedData)
	}
}

func (o *rpcObject) validateRPCInputByModuleName(
	module, name string,
	input string,
) (bool, error) {
	in := map[string]interface{}{
		yangdModuleName + ":rpc-module-name": module,
		yangdModuleName + ":rpc-name":        name,
		yangdModuleName + ":rpc-input":       input,
	}

	var result map[string]interface{}
	err := o.client.Call(yangdModuleName, "validate-rpc-input", in).
		StoreOutputInto(&result)
	if err != nil {
		return false, err
	}
	return result[yangdModuleName+":valid"].(bool), nil
}

// wrapperObject is used to scope common functions used by all the wrappers
type wrapperObject struct {
	client *Client
}

func (o *wrapperObject) validateSet(method reflect.Value) error {
	methodType := method.Type()
	if methodType.NumIn() != 1 {
		return errors.New(
			"Set must have one and only one argument")
	}
	if methodType.NumOut() != 1 {
		return errors.New(
			"Set must have one and only one return value")
	}
	return nil
}

func (o *wrapperObject) validateGet(method reflect.Value) error {
	methodType := method.Type()
	if methodType.NumIn() != 0 {
		return errors.New(
			"Get must have no arguments")
	}
	if methodType.NumOut() != 1 {
		return errors.New(
			"Get must have one and only one return value")
	}
	return nil
}

func (o *wrapperObject) validateCheck(method reflect.Value) error {
	methodType := method.Type()
	if methodType.NumIn() != 1 {
		return errors.New(
			"Check must have one and only one argument")
	}
	if methodType.NumOut() != 1 {
		return errors.New(
			"Check must have one and only one return value")
	}
	return nil
}

func (o *wrapperObject) validateMethod(
	objectVal reflect.Value,
	name string,
	validateFunc func(reflect.Value) error,
) error {
	methodFunc := objectVal.MethodByName(name)
	if !methodFunc.IsValid() {
		return errors.New("Object contains no " +
			name + " method")
	}
	return validateFunc(methodFunc)
}

func (o *wrapperObject) setMethod(
	methodPtr *reflect.Value,
	objectVal reflect.Value,
	name string,
	validateFunc func(reflect.Value) error,
) error {
	err := o.validateMethod(objectVal, name, validateFunc)
	if err != nil {
		return err
	}
	*methodPtr = objectVal.MethodByName(name)
	return nil
}

func (o *wrapperObject) decodeInput(
	typ reflect.Type,
	input string,
) ([]reflect.Value, error) {
	newInput, err := decodeValue(typ, input)
	if err != nil {
		return nil, err
	}
	return []reflect.Value{reflect.ValueOf(newInput)}, nil
}

func (o *wrapperObject) encodeError(err interface{}) error {
	switch out := err.(type) {
	case error:
		return out
	default:
		return nil
	}
}

func (o *wrapperObject) encodeOutput(output interface{}) (string, error) {
	switch v := output.(type) {
	case []byte:
		return string(v), nil
	case string:
		return v, nil
	}

	marshaller := defaultMarshaller()
	return marshaller.Marshal(output)
}
