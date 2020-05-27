// Copyright (c) 2017,2019, AT&T Intellectual Property. All rights reserved.
// Copyright (c) 2016 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package vci

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// Tests

type testConfig struct {
	Value string `rfc7951:"value"`
}

type testRunningConfigWithValue struct {
	testConfig
}

func (c *testRunningConfigWithValue) Get() *testConfig {
	return &testConfig{Value: c.Value}
}
func (c *testRunningConfigWithValue) Set(config *testConfig) error {
	c.Value = config.Value
	return nil
}
func (c *testRunningConfigWithValue) Check(config *testConfig) error {
	return nil
}

type testRunningConfig struct {
}

func (run *testRunningConfig) Get() *testConfig {
	return &testConfig{Value: "foo bar"}
}

func (run *testRunningConfig) Check(config *testConfig) error {
	return nil
}

func (run *testRunningConfig) Set(config *testConfig) error {
	return nil
}

type testRunningConfigNoGet struct {
}

func (run *testRunningConfigNoGet) Check(config *testConfig) error {
	return nil
}

func (run *testRunningConfigNoGet) Set(config *testConfig) error {
	return nil
}

func TestTransportObjectRunningConfigNilWrap(t *testing.T) {
	cfg := newConfig(nil, newClient())
	if cfg.IsValid() {
		t.Error("expected failure creating object got none")
		return
	}
	if cfg.Error() != "cannot wrap nil object" {
		t.Errorf("unexpected error %s", cfg.Error())
	}
}

func TestTransportObjectRunningConfig(t *testing.T) {
	config := newConfig(&testRunningConfig{}, newClient())
	if !config.IsValid() {
		t.Error(config)
		return
	}
}

func TestTransportObjectRunningConfigFunctionMapping(t *testing.T) {
	config := newConfig(&testRunningConfig{}, newClient())
	if !config.IsValid() {
		t.Error(config)
		return
	}
	methods := config.Methods()
	if _, ok := methods["get"]; !ok {
		t.Errorf("function wrapper failed for Get")
	}
	if _, ok := methods["set"]; !ok {
		t.Errorf("function wrapper failed for Set")
	}
	if _, ok := methods["check"]; !ok {
		t.Errorf("function wrapper failed for Check")
	}
}

func TestTransportObjectRunningConfigGetValid(t *testing.T) {
	config := newConfig(&testRunningConfig{}, newClient())
	if !config.IsValid() {
		t.Error(config)
		return
	}
	get := reflect.ValueOf(config.Methods()["get"])
	if get.Kind() != reflect.Func {
		t.Errorf("Get is not a func")
	}
	getType := get.Type()
	if getType.NumIn() != 0 {
		t.Errorf("Get must have no arguments")
	}
	if getType.NumOut() != 2 {
		t.Errorf("Get must have two and only two return values")
	}
}

func TestTransportObjectRunningConfigNoGet(t *testing.T) {
	config := newConfig(&testRunningConfigNoGet{}, newClient())
	if !config.IsValid() {
		t.Error(config)
		return
	}
	_, ok := config.Methods()["get"]
	if ok {
		t.Fatal("Found unexpected get method")
	}
}

func TestConfigName(t *testing.T) {
	config := newConfig(&testRunningConfig{}, newClient())
	if config.Name() != "running" {
		t.Fatal("Name of the config object should be 'running'")
	}
}

func TestConfigType(t *testing.T) {
	config := newConfig(&testRunningConfig{}, newClient())
	if config.Type() != "config" {
		t.Fatal("Type of the config object should be 'config'")
	}
}

type testRunningConfigGetInvalid struct {
	testRunningConfig
}

func (run *testRunningConfigGetInvalid) Get(string) {
}

func TestTransportObjectRunningConfigGetInvalid(t *testing.T) {
	config := newConfig(&testRunningConfigGetInvalid{}, newClient())
	if config.IsValid() {
		t.Error("expected failure creating object got none")
	}

	if config.Error() != "Get must have no arguments" {
		t.Errorf("unexpected error %s", config.Error())
	}
}

type testRunningConfigGetInvalidReturn struct {
	testRunningConfig
}

func (run *testRunningConfigGetInvalidReturn) Get() (*testConfig, error) {
	return nil, nil
}

func TestTransportObjectRunningConfigGetInvalidReturn(t *testing.T) {
	config := newConfig(&testRunningConfigGetInvalidReturn{}, newClient())
	if config.IsValid() {
		t.Error("expected failure creating object got none")
	}
	if config.Error() != "Get must have one and only one return value" {
		t.Errorf("unexpected error %s", config.Error())
	}
}

type testRunningConfigSetInvalid struct {
	testRunningConfig
}

func (run *testRunningConfigSetInvalid) Set() {
}

func TestTransportObjectRunningConfigSetInvalid(t *testing.T) {
	config := newConfig(&testRunningConfigSetInvalid{}, newClient())
	if config.IsValid() {
		t.Error("expected failure creating object got none")
	}

	if config.Error() != "Set must have one and only one argument" {
		t.Errorf("unexpected error %s", config.Error())
	}
}

type testRunningConfigSetInvalidReturn struct {
	testRunningConfig
}

func (run *testRunningConfigSetInvalidReturn) Set(string) (*testConfig, error) {
	return nil, nil
}

func TestTransportObjectRunningConfigSetInvalidReturn(t *testing.T) {
	config := newConfig(&testRunningConfigSetInvalidReturn{}, newClient())
	if config.IsValid() {
		t.Error("expected failure creating object got none")
	}

	if config.Error() != "Set must have one and only one return value" {
		t.Errorf("unexpected error %s", config)
	}
}

type testRunningConfigCheckInvalid struct {
	testRunningConfig
}

func (run *testRunningConfigCheckInvalid) Check() {
}

func TestTransportObjectRunningConfigCheckInvalid(t *testing.T) {
	config := newConfig(&testRunningConfigCheckInvalid{}, newClient())
	if config.IsValid() {
		t.Error("expected failure creating object got none")
	}

	if config.Error() != "Check must have one and only one argument" {
		t.Errorf("unexpected error %s", config.Error())
	}
}

type testRunningConfigCheckInvalidReturn struct {
	testRunningConfig
}

func (run *testRunningConfigCheckInvalidReturn) Check(string) (*testConfig, error) {
	return nil, nil
}

func TestTransportObjectRunningConfigCheckInvalidReturn(t *testing.T) {
	config := newConfig(&testRunningConfigCheckInvalidReturn{}, newClient())
	if config.IsValid() {
		t.Error("expected failure creating object got none")
	}

	if config.Error() != "Check must have one and only one return value" {
		t.Errorf("unexpected error %s", config.Error())
	}
}

func TestTransportObjectRunningConfigGetCall(t *testing.T) {
	config := newConfig(&testRunningConfig{}, newClient())
	if !config.IsValid() {
		t.Error(config)
		return
	}
	get := config.Methods()["get"].(func() (string, error))
	out, err := get()
	if err != nil {
		t.Fatal(err)
	}
	expected := "{\"value\":\"foo bar\"}"
	if out != expected {
		t.Errorf("expected (%s) but got (%s)", expected, out)
	}
}

func TestTransportObjectRunningConfigSetCall(t *testing.T) {
	config := newConfig(&testRunningConfig{}, newClient())
	if !config.IsValid() {
		t.Error(config)
		return
	}
	set := config.Methods()["set"].(func(string) error)
	err := set("{\"value\":\"foo\"}\n")
	if err != nil {
		t.Error(err)
	}
}

type testRunningConfigSetObject struct {
	testRunningConfig
}

func (run *testRunningConfigSetObject) Set(c testConfig) error {
	return nil
}

func TestTransportObjectRunningConfigSetObjectCall(t *testing.T) {
	config := newConfig(&testRunningConfigSetObject{}, newClient())
	if !config.IsValid() {
		t.Error(config)
		return
	}
	set := config.Methods()["set"].(func(string) error)
	err := set("{\"value\":\"foo\"}\n")
	if err != nil {
		t.Error(err)
	}
}

func TestTransportObjectRunningConfigSetObjectInvalidCall(t *testing.T) {
	config := newConfig(&testRunningConfigSetObject{}, newClient())
	if !config.IsValid() {
		t.Error(config)
		return
	}
	set := config.Methods()["set"].(func(string) error)
	err := set("{\"value\":\"foo}\n")
	if err == nil {
		t.Error("method should have failed")
	}
}

type testRunningConfigSetString struct {
	testRunningConfig
}

func (run *testRunningConfigSetString) Set(s string) error {
	return nil
}

func TestTransportObjectRunningConfigSetStringCall(t *testing.T) {
	config := newConfig(&testRunningConfigSetString{}, newClient())
	if !config.IsValid() {
		t.Error(config)
		return
	}
	set := config.Methods()["set"].(func(string) error)
	err := set("{\"value\":\"foo\"}\n")
	if err != nil {
		t.Error(err)
	}
}

type testRunningConfigSetByte struct {
	testRunningConfig
}

func (run *testRunningConfigSetByte) Set(buf string) error {
	return nil
}

func TestTransportObjectRunningConfigSetByteCall(t *testing.T) {
	config := newConfig(&testRunningConfigSetByte{}, newClient())
	if !config.IsValid() {
		t.Error(config)
		return
	}
	set := config.Methods()["set"].(func(string) error)
	err := set("{\"value\":\"foo\"}\n")
	if err != nil {
		t.Error(err)
	}
}

type testRunningConfigSetErrorReturn struct {
	testRunningConfig
}

func (run *testRunningConfigSetErrorReturn) Set(c *testConfig) error {
	return errors.New("c.value = " + c.Value)
}

func TestTransportObjectRunningConfigSetErrorReturnCall(t *testing.T) {
	config := newConfig(&testRunningConfigSetErrorReturn{}, newClient())
	if !config.IsValid() {
		t.Error(config)
		return
	}
	set := config.Methods()["set"].(func(string) error)
	err := set("{\"value\":\"foo\"}\n")
	if err != nil {
		if err.Error() != "c.value = foo" {
			t.Error(err)
		}
	}
}

func TestTransportObjectRunningConfigCheckCall(t *testing.T) {
	config := newConfig(&testRunningConfig{}, newClient())
	if !config.IsValid() {
		t.Error(config)
		return
	}
	check := config.Methods()["check"].(func(string) error)
	err := check("{\"value\":\"foo\"}\n")
	if err != nil {
		t.Error(err)
	}
}

type testRunningConfigCheckObject struct {
	testRunningConfig
}

func (run *testRunningConfigCheckObject) Check(c testConfig) error {
	return nil
}

func TestTransportObjectRunningConfigCheckObjectCall(t *testing.T) {
	config := newConfig(&testRunningConfigCheckObject{}, newClient())
	if !config.IsValid() {
		t.Error(config)
		return
	}
	check := config.Methods()["check"].(func(string) error)
	err := check("{\"value\":\"foo\"}\n")
	if err != nil {
		t.Error(err)
	}
}

func TestTransportObjectRunningConfigCheckObjectInvalidCall(t *testing.T) {
	config := newConfig(&testRunningConfigCheckObject{}, newClient())
	if !config.IsValid() {
		t.Error(config)
		return
	}
	check := config.Methods()["check"].(func(string) error)
	err := check("{\"value\":\"foo}\n")
	if err == nil {
		t.Error("method should have failed")
	}
}

type testRunningConfigCheckString struct {
	testRunningConfig
}

func (run *testRunningConfigCheckString) Check(s string) error {
	return nil
}

func TestTransportObjectRunningConfigCheckStringCall(t *testing.T) {
	config := newConfig(&testRunningConfigCheckString{}, newClient())
	if !config.IsValid() {
		t.Error(config)
		return
	}
	check := config.Methods()["check"].(func(string) error)
	err := check("{\"value\":\"foo\"}\n")
	if err != nil {
		t.Error(err)
	}
}

type testRunningConfigCheckByte struct {
	testRunningConfig
}

func (run *testRunningConfigCheckByte) Check(buf []byte) error {
	return nil
}

func TestTransportObjectRunningConfigCheckByteCall(t *testing.T) {
	config := newConfig(&testRunningConfigCheckByte{}, newClient())
	if !config.IsValid() {
		t.Error(config)
		return
	}
	check := config.Methods()["check"].(func(string) error)
	err := check("{\"value\":\"foo\"}\n")
	if err != nil {
		t.Error(err)
	}
}

type testRunningConfigCheckErrorReturn struct {
	testRunningConfig
}

func (run *testRunningConfigCheckErrorReturn) Set(c *testConfig) error {
	return errors.New("c.value = " + c.Value)
}

func TestTransportObjectRunningConfigCheckErrorReturnCall(t *testing.T) {
	config := newConfig(&testRunningConfigCheckErrorReturn{}, newClient())
	if !config.IsValid() {
		t.Error(config)
		return
	}
	check := config.Methods()["check"].(func(string) error)
	err := check("{\"value\":\"foo\"}\n")
	if err != nil {
		if err.Error() != "c.value = foo" {
			t.Error(err)
		}
	}
}

type testState struct {
	Value string `rfc7951:"value"`
}

func (run *testState) Get() *testState {
	return run
}

func TestTransportObjectConfigWithoutSet(t *testing.T) {
	obj := newConfig(&testState{"foo"}, newClient())
	if obj.IsValid() {
		t.Error("State object as config object should fail")
	}
}

func TestTransportObjectStateGetValid(t *testing.T) {
	obj := newState(&testState{"foo"}, newClient())
	if !obj.IsValid() {
		t.Error(obj)
		return
	}
	get := reflect.ValueOf(obj.Methods()["get"].(func() (string, error)))
	if get.Kind() != reflect.Func {
		t.Errorf("Get is not a func")
	}
	getType := get.Type()
	if getType.NumIn() != 0 {
		t.Errorf("Get must have one and only one argument")
	}
	if getType.NumOut() != 2 {
		t.Errorf("Get must have two and only two return values")
	}
}

func TestTransportObjectStateGetCall(t *testing.T) {
	obj := newState(&testState{"foo"}, newClient())
	if !obj.IsValid() {
		t.Error(obj)
		return
	}
	get := obj.Methods()["get"].(func() (string, error))
	out, err := get()
	if err != nil {
		t.Fatal(err)
	}
	expected := "{\"value\":\"foo\"}"
	if out != expected {
		t.Errorf("expected (%s) but got (%s)", expected, out)
	}
}

func TestTransportObjectStateName(t *testing.T) {
	obj := newState(&testState{"foo"}, newClient())
	if obj.Name() != "state" {
		t.Fatal("Name of the state object should be 'state'")
	}
}

func TestTransportObjectStateType(t *testing.T) {
	obj := newState(&testState{"foo"}, newClient())
	if obj.Type() != "state" {
		t.Fatal("Type of the state object should be 'state'")
	}
}

type testStateGetInvalid struct {
	testState
}

func (run *testStateGetInvalid) Get(string) {
}

func TestTransportObjectStateGetInvalid(t *testing.T) {
	obj := newState(&testStateGetInvalid{testState{"foo"}}, newClient())
	if obj.IsValid() {
		t.Error("expected failure creating object got none")
	}
	if obj.Error() != "Get must have no arguments" {
		t.Errorf("unexpected error %s", obj.Error())
	}
}

type testStateGetInvalidReturn struct {
	testState
}

func (run *testStateGetInvalidReturn) Get() (*testConfig, error) {
	return nil, nil
}

func TestTransportObjectStateGetInvalidReturn(t *testing.T) {
	obj := newState(&testStateGetInvalidReturn{testState{"foo"}},
		newClient())
	if obj.IsValid() {
		t.Error("expected failure creating object got none")
	}
	if obj.Error() != "Get must have one and only one return value" {
		t.Errorf("unexpected error %s", obj.Error())
	}
}

type testRPCs struct{}

func (t *testRPCs) CallMe(in *testConfig) (*testConfig, error) {
	return in, nil
}
func (t *testRPCs) CallMeFail(in *testConfig) (*testConfig, error) {
	return nil, errors.New("broken")
}

type testRPCsInvalidSecondReturn struct {
	testRPCs
}

func (t *testRPCsInvalidSecondReturn) Bogus(in *testConfig) (int, int) {
	return 0, 0
}

func TestTransportObjectRPCInvalidSecondReturn(t *testing.T) {
	obj := newRPC("", &testRPCsInvalidSecondReturn{}, newClient())
	if obj.IsValid() {
		t.Errorf("object should not be valid")
	}
	checkErrorContains(t, obj,
		"All RPCs must have the error type as the second return")
}
func TestTransportObjectRPCNilWrap(t *testing.T) {
	obj := newRPC("", nil, newClient())
	if obj.IsValid() {
		t.Error("expected failure creating object got none")
		return
	}
	if obj.Error() != "cannot wrap nil object" {
		t.Errorf("unexpected error %s", obj.Error())
	}
}

func TestTransportObjectRPCType(t *testing.T) {
	obj := newRPC("", &testRPCs{}, nil)
	if obj.Type() != "rpc" {
		t.Fatal("RPC object type should be 'rpc'")
	}
}

func TestTransportObjectRPC(t *testing.T) {
	rpcs := testGetRPCMethods(t, "", &testRPCs{})

	checkRPCExistsAndIsWrapped(t, rpcs, "call-me")

	checkRPCExistsAndIsWrapped(t, rpcs, "call-me-fail")
}

func TestTransportObjectRPCCall(t *testing.T) {
	rpcs := testGetRPCMethods(t, "", &testRPCs{})

	fn := checkRPCExistsAndIsWrapped(t, rpcs, "call-me")

	in := "{\"value\":\"foo\"}"
	expOut := in
	checkMethodCallSucceeds(t, "CallMe", fn, in, expOut)
}

func TestTransportObjectRPCCallFail(t *testing.T) {
	rpcs := testGetRPCMethods(t, "", &testRPCs{})

	fn := checkRPCExistsAndIsWrapped(t, rpcs, "call-me-fail")

	in := "{\"value\":\"foo\"}"
	expErr := "broken"
	checkMethodCallFails(t, "CallMeFail", fn, in, expErr)
}

func TestTransportObjectRPCInvalidCallFail(t *testing.T) {
	rpcs := testGetRPCMethods(t, "", &testRPCs{})

	fn := checkRPCExistsAndIsWrapped(t, rpcs, "call-me")

	in := "{\"value\":\"foo}"
	expErr := "unexpected end of JSON input"
	checkMethodCallFails(t, "CallMe", fn, in, expErr)
}

func callMeDirect(in *testConfig) (*testConfig, error) {
	return in, nil
}
func callMeFailDirect(in *testConfig) (*testConfig, error) {
	return nil, errors.New("broken")
}
func callMeStrings(in string) (string, error) {
	return in, nil
}

func TestTransportObjectRPCAsMap(t *testing.T) {
	rpcMap := make(map[string]interface{})
	rpcMap["CallMeDirect"] = callMeDirect
	rpcMap["CallMeFailDirect"] = callMeFailDirect

	rpcs := testGetRPCMethods(t, "", rpcMap)

	checkRPCExistsAndIsWrapped(t, rpcs, "CallMeDirect")

	checkRPCExistsAndIsWrapped(t, rpcs, "CallMeFailDirect")
}

func TestTransportObjectRPCMapSkipsNonFunction(t *testing.T) {
	rpcMap := make(map[string]interface{})
	rpcMap["CallMeDirect"] = callMeDirect
	rpcMap["NotAFunction"] = "foobar"
	rpcs := testGetRPCMethods(t, "", rpcMap)

	checkRPCExistsAndIsWrapped(t, rpcs, "CallMeDirect")
	checkRPCDoesNotExist(t, rpcs, "NotAFunction")

}

func TestTransportObjectRPCMapSkipsWrongNumIn(t *testing.T) {
	rpcMap := make(map[string]interface{})
	rpcMap["CallMeDirect"] = callMeDirect
	rpcMap["NotValid"] = func(a, b int) (int, error) {
		return 0, nil
	}
	rpcs := testGetRPCMethods(t, "", rpcMap)

	checkRPCExistsAndIsWrapped(t, rpcs, "CallMeDirect")
	checkRPCDoesNotExist(t, rpcs, "NotValid")

}
func TestTransportObjectRPCMapSkipsWrongNumOut(t *testing.T) {
	rpcMap := make(map[string]interface{})
	rpcMap["CallMeDirect"] = callMeDirect
	rpcMap["NotValid"] = func(a int) int {
		return 0
	}
	rpcs := testGetRPCMethods(t, "", rpcMap)

	checkRPCExistsAndIsWrapped(t, rpcs, "CallMeDirect")
	checkRPCDoesNotExist(t, rpcs, "NotValid")

}
func TestTransportObjectRPCMapSkipsWrongOutType(t *testing.T) {
	rpcMap := make(map[string]interface{})
	rpcMap["CallMeDirect"] = callMeDirect
	rpcMap["NotValid"] = func(a int) (int, int) {
		return 0, 0
	}
	rpcs := testGetRPCMethods(t, "", rpcMap)

	checkRPCExistsAndIsWrapped(t, rpcs, "CallMeDirect")
	checkRPCDoesNotExist(t, rpcs, "NotValid")

}

func TestTransportObjectRPCCallAsMap(t *testing.T) {
	rpcMap := make(map[string]interface{})
	rpcMap["CallMeDirect"] = callMeDirect
	rpcMap["CallMeFailDirect"] = callMeFailDirect

	rpcs := testGetRPCMethods(t, "", rpcMap)

	fn := checkRPCExistsAndIsWrapped(t, rpcs, "CallMeDirect")

	in := "{\"value\":\"foo\"}"
	expOut := in
	checkMethodCallSucceeds(t, "CallMeDirect", fn, in, expOut)
}

func TestTransportObjectRPCStrings(t *testing.T) {
	rpcMap := make(map[string]interface{})
	rpcMap["call-me-strings"] = callMeStrings
	rpcs := testGetRPCMethods(t, "", rpcMap)
	fn := checkRPCExistsAndIsWrapped(t, rpcs, "call-me-strings")
	in := "{\"value\":\"foo\"}"
	expOut := in
	checkMethodCallSucceeds(t, "call-me-strings", fn, in, expOut)
}

func TestTransportObjectRPCBytes(t *testing.T) {
	rpcMap := make(map[string]interface{})
	rpcMap["call-me-bytes"] = func(in []byte) ([]byte, error) {
		return in, nil
	}
	rpcs := testGetRPCMethods(t, "", rpcMap)
	fn := checkRPCExistsAndIsWrapped(t, rpcs, "call-me-bytes")
	checkRPCExistsAndIsWrapped(t, rpcs, "call-me-bytes")
	in := "{\"value\":\"foo\"}"
	expOut := in
	checkMethodCallSucceeds(t, "call-me-bytes", fn, in, expOut)
}

func TestTransportObjectRPCMap(t *testing.T) {
	rpcMap := make(map[string]interface{})
	rpcMap["call-me-map"] = func(in map[string]interface{}) (map[string]interface{}, error) {
		return in, nil
	}
	rpcs := testGetRPCMethods(t, "", rpcMap)
	fn := checkRPCExistsAndIsWrapped(t, rpcs, "call-me-map")
	checkRPCExistsAndIsWrapped(t, rpcs, "call-me-map")
	in := "{\"value\":\"foo\"}"
	expOut := in
	checkMethodCallSucceeds(t, "call-me-map", fn, in, expOut)
}

func TestTransportObjectRPCChanIntIsInvalid(t *testing.T) {
	rpcMap := make(map[string]interface{})
	rpcMap["call-me-slice"] = func(in string) (chan int, error) {
		return make(chan int), nil
	}
	rpcs := testGetRPCMethods(t, "", rpcMap)
	fn := checkRPCExistsAndIsWrapped(t, rpcs, "call-me-slice")
	checkRPCExistsAndIsWrapped(t, rpcs, "call-me-slice")
	in := "{\"value\":\"foo\"}"
	expOut := "json: unsupported type: chan int"
	checkMethodCallFails(t, "call-me-slice", fn, in, expOut)
}

type testRPCIncorrectNumArgs struct {
	testRPCs
}

func (t *testRPCIncorrectNumArgs) Fail() (*testConfig, error) {
	return nil, nil
}

func checkErrorContains(t *testing.T, err error, exp string) {
	if err == nil {
		t.Fatalf("Expected failure didn't occur")
		return
	}
	if len(exp) == 0 {
		t.Fatalf("Expected output must have non-zero length.")
		return
	}

	if !strings.Contains(err.Error(), exp) {
		t.Fatalf("Actual error doesn't have expected content:\n"+
			"Exp:\n%s\nAct:\n%v\n", exp, err.Error())
	}
}

func TestTransportObjectRPCIncorrectNumArgs(t *testing.T) {
	obj := newRPC("", &testRPCIncorrectNumArgs{}, newClient())
	if obj.IsValid() {
		t.Errorf("object should not be valid")
	}
	checkErrorContains(t, obj,
		"All RPCs must have one and only one argument")
}

type testRPCIncorrectNumReturn struct {
	testRPCs
}

func (t *testRPCIncorrectNumReturn) Fail(*testConfig) error {
	return nil
}

func TestTransportObjectRPCIncorrectNumReturn(t *testing.T) {
	obj := newRPC("", &testRPCIncorrectNumReturn{}, newClient())
	if obj.IsValid() {
		t.Errorf("object should not be valid")
	}
	checkErrorContains(t, obj,
		"All RPCs must have 2 and only 2 returns")
}

// Test validateInput.  Here we 'mock' the bus call to yangd to validate
// the RPC against the schema.
type testValidateRPCs struct{}

func (t *testValidateRPCs) ValidateMe(in *testConfig) (*testConfig, error) {
	return in, nil
}

type testTORPCPromise string

func (t testTORPCPromise) StoreOutputInto(out *string) error {
	*out = string(t)
	return nil
}

//testYangdRpc is good enough for testing these functions but
//a better mock bus is needed for more involved tests.
type testYangdRPC struct {
	transporter
	validateInputJSON string
	desiredResult     bool
}

func newTestYangdRPC(desiredResult bool) *testYangdRPC {
	return &testYangdRPC{
		transporter:   nil,
		desiredResult: desiredResult,
	}
}

func (yr *testYangdRPC) Call(
	moduleName, rpcName, rpcReq string,
) (transportRPCPromise, error) {
	yr.validateInputJSON = rpcReq

	switch yr.desiredResult {
	case true:
		return testTORPCPromise("{\"yangd-v1:valid\":true}"), nil
	case false:
		return testTORPCPromise("{\"yangd-v1:valid\":false}"),
			errors.New("ValIn FAIL")
	}
	return testTORPCPromise(""), errors.New("Undefined result")
}

func TestWrapValidateInputPass(t *testing.T) {
	tYR := newTestYangdRPC(true)
	rpcs := testGetRPCMethodsCustomTransport(
		t, "test-module", &testValidateRPCs{}, tYR)

	fn := checkRPCExistsAndIsWrapped(t, rpcs, "validate-me")

	in := "{\"value\":\"foo\"}"
	expOut := in
	checkMethodCallSucceeds(t, "ValidateMe", fn, in, expOut)

	expData := map[string]interface{}{
		yangdModuleName + ":rpc-module-name": "test-module",
		yangdModuleName + ":rpc-name":        "validate-me",
		yangdModuleName + ":rpc-input":       "{\"value\":\"foo\"}",
	}
	data, err := testGenericallyUnmarshalTestData(tYR.validateInputJSON)
	if err != nil {
		t.Fatal(err)
	}
	if !testCompareMaps(expData, data) {
		t.Fatalf("Got: %s\n", tYR.validateInputJSON)
	}
}

func TestWrapValidateInputFail(t *testing.T) {
	tYR := newTestYangdRPC(false)
	rpcs := testGetRPCMethodsCustomTransport(
		t, "test-model", &testValidateRPCs{}, tYR)

	fn := checkRPCExistsAndIsWrapped(t, rpcs, "validate-me")

	in := "{\"value\":\"foo\"}"
	checkMethodCallFails(t, "ValidateMe", fn, in, "ValIn FAIL")
}

func TestPassthrough(t *testing.T) {
	t.Skipf("Plenty to test here ...")
}

// Helper functions
func testCompareMaps(expected, actual map[string]interface{}) bool {
	//ensure actual is a superset of expected
	for name, expVal := range expected {
		val, ok := actual[name]
		if !ok {
			return false
		}
		if val != expVal {
			return false
		}
	}
	return true
}

func testGenericallyUnmarshalTestData(in string) (map[string]interface{}, error) {
	marshaller := defaultMarshaller()
	var out map[string]interface{}
	err := marshaller.Unmarshal(in, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func testGetRPCMethodsCustomTransport(
	t *testing.T,
	moduleName string,
	object interface{},
	transport transporter,
) map[string]interface{} {

	client := newClient().withTransport(transport)
	o := newRPC(moduleName, object, client)
	if !o.IsValid() {
		t.Error(o)
	}
	return o.Methods()
}

func testGetRPCMethods(
	t *testing.T,
	modelName string,
	object interface{},
) map[string]interface{} {
	return testGetRPCMethodsCustomTransport(t, modelName, object,
		defaultTransport())
}

func checkRPCExistsAndIsWrapped(
	t *testing.T,
	rpcs map[string]interface{},
	methodName string,
) func(string) (string, error) {

	method, exists := rpcs[methodName]
	if !exists {
		t.Fatalf("Expected %s method doesn't exist\n", methodName)
		return nil
	}
	fn, ok := method.(func(string) (string, error))
	if !ok {
		t.Fatalf("%s is not wrapped", methodName)
		return nil
	}

	return fn
}

func checkRPCDoesNotExist(
	t *testing.T,
	rpcs map[string]interface{},
	methodName string,
) {
	_, exists := rpcs[methodName]
	if exists {
		t.Fatalf("Expected %s method exists and should not\n", methodName)
	}
}

func checkMethodCallSucceeds(
	t *testing.T,
	methodName string,
	fn func(string) (string, error),
	in string,
	expOut string,
) {
	out, err := fn(in)
	if err != nil {
		t.Error(err)
	}
	if expOut != string(out) {
		t.Errorf("Called %s()\nExp:\n\t%s\nGot:\n\t%s",
			methodName, expOut, string(out))
	}
}

func checkMethodCallFails(
	t *testing.T,
	methodName string,
	fn func(string) (string, error),
	in string,
	expErr string,
) {
	_, err := fn(in)
	if err == nil {
		t.Fatalf("Unexpected success running %s()", methodName)
		return
	}
	if err.Error() != expErr {
		t.Errorf("\nExp:\n\t%s\nGot:\n\t%s", expErr, err.Error())
	}

}
