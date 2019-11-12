// Copyright (c) 2017-2019, AT&T Intellectual Property. All rights reserved.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package vci

import (
	"testing"
	"time"
)

func CallMeOne(in *testConfig) (*testConfig, error) {
	if in == nil {
		in = new(testConfig)
	}
	in.Value = "One"
	return in, nil
}

func CallMeTwo(in *testConfig) (*testConfig, error) {
	if in == nil {
		in = new(testConfig)
	}
	in.Value = "Two"
	return in, nil
}

func TestComponent(t *testing.T) {
	t.Run("component-supports-multiple-models", func(t *testing.T) {
		resetTestBus()
		comp := NewComponent("net.vyatta.test")
		comp.Model("net.vyatta.test.v1").
			Config(&testRunningConfigWithValue{
				testConfig{Value: "One"}}).
			State(&testState{Value: "One"}).
			RPC("test-v1", map[string]interface{}{
				"call-me": CallMeOne,
			})
		comp.Model("net.vyatta.test.v2").
			Config(&testRunningConfigWithValue{
				testConfig{Value: "Two"}}).
			State(&testState{Value: "Two"}).
			RPC("test-v2", map[string]interface{}{
				"call-me": CallMeTwo,
			})
		err := comp.Run()
		if err != nil {
			t.Fatal(err)
		}

		client, err := Dial()
		if err != nil {
			t.Fatal(err)
		}
		t.Run("config", func(t *testing.T) {
			var out map[string]interface{}
			err = client.StoreConfigByModelInto(
				"net.vyatta.test.v1",
				&out)
			if err != nil {
				t.Fatal(err)
			}
			if out["value"] != "One" {
				t.Fatal(
					"value for net.vyatta.test.v1 wasn't received")
			}

			err = client.StoreConfigByModelInto(
				"net.vyatta.test.v2",
				&out)
			if err != nil {
				t.Fatal(err)
			}
			if out["value"] != "Two" {
				t.Fatal(
					"value for net.vyatta.test.v2 wasn't received")
			}
		})
		t.Run("state", func(t *testing.T) {
			var out map[string]interface{}
			err = client.StoreStateByModelInto(
				"net.vyatta.test.v1",
				&out)
			if err != nil {
				t.Fatal(err)
			}
			if out["value"] != "One" {
				t.Fatal(
					"value for net.vyatta.test.v1 wasn't received")
			}

			err = client.StoreStateByModelInto(
				"net.vyatta.test.v2",
				&out)
			if err != nil {
				t.Fatal(err)
			}
			if out["value"] != "Two" {
				t.Fatal(
					"value for net.vyatta.test.v2 wasn't received")
			}
		})
		t.Run("RPC", func(t *testing.T) {
			var out map[string]interface{}
			err = client.Call("test-v1", "call-me",
				map[string]interface{}{}).
				StoreOutputInto(&out)
			if err != nil {
				t.Fatal(err)
			}
			if out["value"] != "One" {
				t.Fatal("value for CallMeOne wasn't received")
			}

			err = client.Call("test-v2", "call-me",
				map[string]interface{}{}).
				StoreOutputInto(&out)
			if err != nil {
				t.Fatal(err)
			}
			if out["value"] != "Two" {
				t.Fatal("value for CallMeTwo wasn't received")
			}
		})
	})
	t.Run("component-can-subscribe-to-notifications", func(t *testing.T) {
		resetTestBus()
		comp := NewComponent("net.vyatta.test")
		comp.Model("net.vyatta.test.v1").
			Config(&testRunningConfigWithValue{
				testConfig{Value: "One"}}).
			State(&testState{Value: "One"}).
			RPC("test-v1", map[string]interface{}{
				"call-me": CallMeOne,
			})
		done := make(chan struct{})
		err := comp.Subscribe("foo-v1", "bar",
			func(in map[string]interface{}) {
				if in["baz"] != "quux" {
					t.Fatalf(
						"expected %q, got %q\n",
						"quux",
						in["baz"])
				}
				close(done)
			})
		if err != nil {
			t.Fatal(err)
		}
		err = comp.Run()
		if err != nil {
			t.Fatal(err)
		}
		client, err := Dial()
		if err != nil {
			t.Fatal(err)
		}
		err = client.Emit("foo-v1", "bar", map[string]interface{}{
			"baz": "quux",
		})
		if err != nil {
			t.Fatal(err)
		}
		select {
		case <-done:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Notification didn't arrive")
		}
	})
	t.Run("component-can-unsubscribe-from-notification",
		func(t *testing.T) {
			resetTestBus()
			comp := NewComponent("net.vyatta.test")
			comp.Model("net.vyatta.test.v1").
				Config(&testRunningConfigWithValue{
					testConfig{Value: "One"}}).
				State(&testState{Value: "One"}).
				RPC("test-v1", map[string]interface{}{
					"call-me": CallMeOne,
				})
			done := make(chan struct{})
			err := comp.Subscribe("foo-v1", "bar",
				func(in map[string]interface{}) {
					if in["baz"] != "quux" {
						t.Fatalf(
							"expected %q, got %q\n",
							"quux",
							in["baz"])
					}
					done <- struct{}{}
				})
			if err != nil {
				t.Fatal(err)
			}

			err = comp.Run()
			if err != nil {
				t.Fatal(err)
			}

			client, err := Dial()
			if err != nil {
				t.Fatal(err)
			}

			err = client.Emit("foo-v1", "bar",
				map[string]interface{}{
					"baz": "quux",
				})
			if err != nil {
				t.Fatal(err)
			}

			select {
			case <-done:
			case <-time.After(100 * time.Millisecond):
				t.Fatal("Notification didn't arrive")
			}

			err = comp.Unsubscribe("foo-v1", "bar")
			if err != nil {
				t.Fatal(err)
			}

			err = client.Emit("foo-v1", "bar",
				map[string]interface{}{
					"baz": "quux",
				})
			if err != nil {
				t.Fatal(err)
			}

			select {
			case <-done:
				t.Fatal("Notification didn't arrive")
			case <-time.After(100 * time.Millisecond):
			}

		})
	t.Run("component-can-provide-subscription-on-lookup",
		func(t *testing.T) {
			resetTestBus()
			comp := NewComponent("net.vyatta.test")
			comp.Model("net.vyatta.test.v1").
				Config(&testRunningConfigWithValue{
					testConfig{Value: "One"}}).
				State(&testState{Value: "One"}).
				RPC("test-v1", map[string]interface{}{
					"call-me": CallMeOne,
				})
			done := make(chan struct{})
			err := comp.Subscribe("foo-v1", "bar",
				func(in map[string]interface{}) {
					if in["baz"] != "quux" {
						t.Fatalf(
							"expected %q, got %q\n",
							"quux",
							in["baz"])
					}
					done <- struct{}{}
				})
			if err != nil {
				t.Fatal(err)
			}
			err = comp.Run()
			if err != nil {
				t.Fatal(err)
			}

			sub := comp.LookupSubscription("foo-v1", "bar")
			if sub == nil {
				t.Fatal("expected subscription not returned")
			}
			sub = comp.LookupSubscription("foo-v1", "baz")
			if sub != nil {
				t.Fatal("unexpected subscription not returned")
			}

		})
	t.Run("component-can-subscribe-while-running", func(t *testing.T) {
		resetTestBus()
		comp := NewComponent("net.vyatta.test")
		comp.Model("net.vyatta.test.v1").
			Config(&testRunningConfigWithValue{
				testConfig{Value: "One"}}).
			State(&testState{Value: "One"}).
			RPC("test-v1", map[string]interface{}{
				"call-me": CallMeOne,
			})
		err := comp.Run()
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan struct{})
		err = comp.Subscribe("foo-v1", "bar",
			func(in map[string]interface{}) {
				if in["baz"] != "quux" {
					t.Fatalf(
						"expected %q, got %q\n", "quux",
						in["baz"])
				}
				done <- struct{}{}
			})
		if err != nil {
			t.Fatal(err)
		}

		client, err := Dial()
		if err != nil {
			t.Fatal(err)
		}

		err = client.Emit("foo-v1", "bar",
			map[string]interface{}{
				"baz": "quux",
			})
		if err != nil {
			t.Fatal(err)
		}

		select {
		case <-done:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Notification didn't arrive")
		}

	})
	t.Run("component-fails-when-transport-is-broken",
		func(t *testing.T) {
			resetTestBus()
			tBus.toggleDialFailure()
			comp := NewComponent("net.vyatta.test")
			comp.Model("net.vyatta.test.v1").
				Config(&testRunningConfigWithValue{
					testConfig{Value: "One"}}).
				State(&testState{Value: "One"}).
				RPC("test-v1", map[string]interface{}{
					"call-me": CallMeOne,
				})
			err := comp.Run()
			if err == nil {
				t.Fatal("expected failure didn't occur")
			}
		})
	t.Run("component-fails-when-exporting-bogus-models",
		func(t *testing.T) {
			t.Run("config", func(t *testing.T) {
				resetTestBus()
				comp := NewComponent("net.vyatta.test")
				comp.Model("net.vyatta.test.v1").
					Config(&testState{Value: "One"}).
					State(&testState{Value: "One"}).
					RPC("test-v1", map[string]interface{}{
						"call-me": CallMeOne,
					})
				err := comp.Run()
				if err == nil {
					t.Fatal("expected failure didn't occur")
				}
			})
			t.Run("state", func(t *testing.T) {
				resetTestBus()
				comp := NewComponent("net.vyatta.test")
				comp.Model("net.vyatta.test.v1").
					Config(&testRunningConfigWithValue{
						testConfig{Value: "One"}}).
					State(&testStateGetInvalid{}).
					RPC("test-v1", map[string]interface{}{
						"call-me": CallMeOne,
					})
				err := comp.Run()
				if err == nil {
					t.Fatal("expected failure didn't occur")
				}
			})
			t.Run("rpc", func(t *testing.T) {
				resetTestBus()
				comp := NewComponent("net.vyatta.test")
				comp.Model("net.vyatta.test.v1").
					Config(&testRunningConfigWithValue{
						testConfig{Value: "One"}}).
					State(&testState{}).
					RPC("test-v1",
						&testRPCsInvalidSecondReturn{})
				err := comp.Run()
				if err == nil {
					t.Fatal("expected failure didn't occur")
				}
			})
		})
	t.Run("component-client-returns-functional-client", func(t *testing.T) {
		resetTestBus()
		comp := NewComponent("net.vyatta.test")
		comp.Model("net.vyatta.test.v1").
			Config(&testRunningConfigWithValue{
				testConfig{Value: "One"}}).
			State(&testState{Value: "One"}).
			RPC("test-v1", map[string]interface{}{
				"call-me": CallMeOne,
			})
		err := comp.Run()
		if err != nil {
			t.Fatal(err)
		}
		client := comp.Client()
		var out map[string]interface{}
		err = client.Call("test-v1", "call-me",
			map[string]interface{}{
				"value": "Zero",
			}).StoreOutputInto(&out)
		if err != nil {
			t.Fatal(err)
		}
		if out["value"] != "One" {
			t.Fatal("call didn't return expected value")
		}
	})
	t.Run("(*component).Wait blocks until (*component).Stop", func(t *testing.T) {
		resetTestBus()
		comp := NewComponent("net.vyatta.test")
		comp.Model("net.vyatta.test.v1").
			Config(&testRunningConfigWithValue{
				testConfig{Value: "One"}}).
			State(&testState{Value: "One"}).
			RPC("test-v1", map[string]interface{}{
				"call-me": CallMeOne,
			})
		err := comp.Run()
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan struct{})
		go func() {
			comp.Wait()
			close(done)
		}()
		select {
		case <-done:
			t.Fatal("Wait did not block as expected")
		case <-time.After(100 * time.Millisecond):
		}
		comp.Stop()
		select {
		case <-done:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Wait did not terminate as expected")
		}
	})

}

func TestModel(t *testing.T) {
	t.Run("model-can-export-multiple-rpc-modules", func(t *testing.T) {
		resetTestBus()
		comp := NewComponent("net.vyatta.test")
		comp.Model("net.vyatta.test.v1").
			Config(&testRunningConfigWithValue{
				testConfig{Value: "One"}}).
			State(&testState{Value: "One"}).
			RPC("test-v1", map[string]interface{}{
				"call-me": CallMeOne,
			}).
			RPC("test-v2", map[string]interface{}{
				"call-me": CallMeTwo,
			})

		err := comp.Run()
		if err != nil {
			t.Fatal(err)
		}

		client, err := Dial()
		if err != nil {
			t.Fatal(err)
		}
		var out map[string]interface{}
		err = client.Call("test-v1", "call-me",
			map[string]interface{}{}).
			StoreOutputInto(&out)
		if err != nil {
			t.Fatal(err)
		}
		if out["value"] != "One" {
			t.Fatal("value for CallMeOne wasn't received")
		}

		err = client.Call("test-v2", "call-me",
			map[string]interface{}{}).
			StoreOutputInto(&out)
		if err != nil {
			t.Fatal(err)
		}
		if out["value"] != "Two" {
			t.Fatal("value for CallMeTwo wasn't received")
		}
	})
	t.Run("model-can-export-multiple-notification-modules",
		func(t *testing.T) {
			t.Skip("no way to test this, it is DBus specific...")
		})
	t.Run("model-fails-when-transport-is-broken", func(t *testing.T) {
		resetTestBus()
		tBus.toggleDialFailure()
		m := newModel("net.vyatta.test.v1", nil)
		err := m.run()
		if err == nil {
			t.Logf("expected failure did not occur")
		}
	})
	t.Run("model-fails-when-exporting-bogus-config-object",
		func(t *testing.T) {
			resetTestBus()
			m := newModel("net.vyatta.test.v1", nil)
			m.Config(&testState{Value: "One"}).
				State(&testState{Value: "One"}).
				RPC("test-v1", map[string]interface{}{
					"call-me": CallMeOne,
				})
			err := m.run()
			if err == nil {
				t.Fatal("expected failure didn't occur")
			}
		})
	t.Run("model-fails-when-exporting-bogus-state-object",
		func(t *testing.T) {
			resetTestBus()
			m := newModel("net.vyatta.test.v1", nil)
			m.Config(&testRunningConfigWithValue{
				testConfig{Value: "One"}}).
				State(&testStateGetInvalid{}).
				RPC("test-v1", map[string]interface{}{
					"call-me": CallMeOne,
				})
			err := m.run()
			if err == nil {
				t.Fatal("expected failure didn't occur")
			}
		})

	t.Run("model-fails-when-exporting-bogus-rpc-object",
		func(t *testing.T) {
			resetTestBus()
			m := newModel("net.vyatta.test.v1", nil)
			m.Config(&testRunningConfigWithValue{
				testConfig{Value: "One"}}).
				State(&testState{}).
				RPC("test-v1",
					&testRPCsInvalidSecondReturn{})
			err := m.run()
			if err == nil {
				t.Fatal("expected failure didn't occur")
			}
		})

	t.Run("model-fails-when-exporting-bogus-notifications-object",
		func(t *testing.T) {
			t.Skip("no way to test this, it is DBus specific...")
		})
}
