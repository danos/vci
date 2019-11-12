// Copyright (c) 2017,2019, AT&T Intellectual Property. All rights reserved.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package vci

import (
	"testing"
	"time"
)

func TestClientDial(t *testing.T) {
	t.Run("normal-dial", func(t *testing.T) {
		resetTestBus()
		_, err := Dial()
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("failing-dial", func(t *testing.T) {
		resetTestBus()
		_, err := Dial()
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("no-transport", func(t *testing.T) {
		defer func() {
			err := recover().(error)
			if err == nil {
				t.Fatal("expected an error")
			}
		}()
		resetTestBus()
		client := newClient()
		client.dial()
	})
}

func TestClientCallComponent(t *testing.T) {
	resetTestBus()

	comp := NewComponent("com.vyatta.test.foo")
	comp.Model("com.vyatta.test.foo.v1").
		RPC("foo-v1", &testRPCs{})
	err := comp.Run()
	if err != nil {
		t.Fatal(err)
	}

	client, err := Dial()
	if err != nil {
		t.Fatal(err)
	}

	t.Run("Valid-Input", func(t *testing.T) {
		var out map[string]interface{}
		err = client.Call("foo-v1", "call-me",
			map[string]interface{}{
				"value": "foobar",
			}).StoreOutputInto(&out)
		if err != nil {
			t.Fatal(err)
		}

		if out["value"] != "foobar" {
			t.Fatalf("expected %q, got %q",
				"foobar",
				out["value"])
		}
	})
	t.Run("Invalid-Input", func(t *testing.T) {
		var out map[string]interface{}
		err = client.Call("foo-v1", "call-me",
			make(chan struct{})).StoreOutputInto(&out)
		if err == nil {
			t.Fatal(err)
		}

	})

}

func TestClientSubscribe(t *testing.T) {
	t.Run("valid-subscription", func(t *testing.T) {
		resetTestBus()
		client, err := Dial()
		if err != nil {
			t.Fatal(err)
		}

		done := make(chan struct{})
		err = client.Subscribe("foo-v1", "bar",
			func(in map[string]interface{}) {
				if in["baz"] != "quux" {
					t.Fatalf(
						"expected %q, got %q\n", "quux",
						in["baz"])
				}
				close(done)
			}).Run()
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
	t.Run("channel-subsciber", func(t *testing.T) {
		resetTestBus()
		client, err := Dial()
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan struct{})
		ch := make(chan map[string]interface{})
		go func() {
			<-ch
			close(done)
		}()
		err = client.Subscribe("foo-v1", "bar", ch).Run()
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
	t.Run("invalid-subscriber-type", func(t *testing.T) {
		resetTestBus()
		client, err := Dial()
		if err != nil {
			t.Fatal(err)
		}

		err = client.Subscribe("foo-v1", "bar",
			make([]string, 0)).Run()
		if err == nil {
			t.Fatal("expected failure, " +
				"subscribed with invalid type")
		}
	})
	t.Run("invalid-subscription", func(t *testing.T) {
		resetTestBus()
		client, err := Dial()
		if err != nil {
			t.Fatal(err)
		}

		err = client.Subscribe("foo-v1", "bar",
			func(foo, bar string) {
			}).Run()
		if err == nil {
			t.Fatal("expected failure, " +
				"subscribed with non unary function")
		}
	})
	t.Run("dial-failure", func(t *testing.T) {
		resetTestBus()
		tBus.toggleDialFailure()
		defer tBus.toggleDialFailure()
		client, _ := Dial()

		err := client.Subscribe("foo-v1", "bar",
			func(foo string) {
			}).Run()
		if err == nil {
			t.Fatal("expected failure, " +
				"dial failed")
		}
	})
	t.Run("emit-invalid-type", func(t *testing.T) {
		resetTestBus()
		client, err := Dial()
		if err != nil {
			t.Fatal(err)
		}
		err = client.Emit("foo-v1", "bar", make(chan struct{}))
		if err == nil {
			t.Fatal("should have failed, invalid type emitted")
		}
	})

}

func TestClientStoreConfigByModelInto(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		resetTestBus()
		comp := NewComponent("com.vyatta.test.foo")
		comp.Model("com.vyatta.test.foo.v1").
			Config(&testRunningConfig{})
		err := comp.Run()
		if err != nil {
			t.Fatal(err)
		}

		client, err := Dial()
		if err != nil {
			t.Fatal(err)
		}

		var out map[string]interface{}
		err = client.StoreConfigByModelInto("com.vyatta.test.foo.v1",
			&out)
		if err != nil {
			t.Fatal(err)
		}

		if out["value"] != "foo bar" {
			t.Fatalf("unexpected config returned")
		}
	})
	t.Run("invalid-return", func(t *testing.T) {
		resetTestBus()
		comp := NewComponent("com.vyatta.test.foo")
		comp.Model("com.vyatta.test.foo.v1").
			Config(&testRunningConfig{})
		err := comp.Run()
		if err != nil {
			t.Fatal(err)
		}
		tBus.toggleDialFailure()
		defer tBus.toggleDialFailure()

		client, _ := Dial()

		var out map[string]interface{}
		err = client.StoreConfigByModelInto("com.vyatta.test.foo.v1",
			&out)
		if err == nil {
			t.Fatal("should have failed, not connected")
		}
	})
	t.Run("invalid-unmarshal-value", func(t *testing.T) {
		resetTestBus()
		comp := NewComponent("com.vyatta.test.foo")
		comp.Model("com.vyatta.test.foo.v1").
			Config(&testRunningConfig{})
		err := comp.Run()
		if err != nil {
			t.Fatal(err)
		}

		client, err := Dial()
		if err != nil {
			t.Fatal(err)
		}

		var out chan struct{}
		err = client.StoreConfigByModelInto("com.vyatta.test.foo.v1",
			&out)
		if err == nil {
			t.Fatal("should have failed, invalid type")
		}
	})

}

func TestClientStoreStateByModelInto(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		resetTestBus()

		comp := NewComponent("com.vyatta.test.foo")
		comp.Model("com.vyatta.test.foo.v1").
			State(&testState{Value: "foo bar"})
		err := comp.Run()
		if err != nil {
			t.Fatal(err)
		}

		client, err := Dial()
		if err != nil {
			t.Fatal(err)
		}

		var out map[string]interface{}
		err = client.StoreStateByModelInto("com.vyatta.test.foo.v1",
			&out)
		if err != nil {
			t.Fatal(err)
		}

		if out["value"] != "foo bar" {
			t.Fatalf("unexpected state returned")
		}
	})
	t.Run("invalid-return", func(t *testing.T) {
		resetTestBus()
		comp := NewComponent("com.vyatta.test.foo")
		comp.Model("com.vyatta.test.foo.v1").
			State(&testState{})
		err := comp.Run()
		if err != nil {
			t.Fatal(err)
		}
		tBus.toggleDialFailure()
		defer tBus.toggleDialFailure()

		client, _ := Dial()

		var out map[string]interface{}
		err = client.StoreStateByModelInto("com.vyatta.test.foo.v1",
			&out)
		if err == nil {
			t.Fatal("should have failed, not connected")
		}
	})
	t.Run("invalid-unmarshal-value", func(t *testing.T) {
		resetTestBus()
		comp := NewComponent("com.vyatta.test.foo")
		comp.Model("com.vyatta.test.foo.v1").
			State(&testState{})
		err := comp.Run()
		if err != nil {
			t.Fatal(err)
		}

		client, err := Dial()
		if err != nil {
			t.Fatal(err)
		}

		var out chan struct{}
		err = client.StoreStateByModelInto("com.vyatta.test.foo.v1",
			&out)
		if err == nil {
			t.Fatal("should have failed, invalid type")
		}
	})
}
