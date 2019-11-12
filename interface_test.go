// Copyright (c) 2017,2019, AT&T Intellectual Property. All rights reserved.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package vci

import (
	"testing"
	"time"
)

func TestCallRPC(t *testing.T) {
	resetTestBus()
	comp := NewComponent("com.vyatta.test.foo")
	comp.Model("com.vyatta.test.foo.v1").
		RPC("foo-v1", &testRPCs{})
	err := comp.Run()
	if err != nil {
		t.Fatal(err)
	}

	t.Run("call-me", func(t *testing.T) {
		input := "{\"value\":\"foobar\"}"
		var output string
		err := CallRPC("foo-v1", "call-me", input).
			StoreOutputInto(&output)
		if err != nil {
			t.Fatal("call-me should have succeeded")
		}
		if output != input {
			t.Fatal("unexpected output from call-me")
		}
	})
	t.Run("call-me-fail", func(t *testing.T) {
		input := "{\"value\":\"foobar\"}"
		var output string
		err := CallRPC("foo-v1", "call-me-fail", input).
			StoreOutputInto(&output)
		if err == nil {
			t.Fatal("call-me-fail should have failed during call")
		}
	})
	t.Run("dial-failed", func(t *testing.T) {
		tBus.toggleDialFailure()
		defer tBus.toggleDialFailure()

		input := "{\"value\":\"foobar\"}"
		var output string
		err := CallRPC("foo-v1", "call-me-fail", input).
			StoreOutputInto(&output)

		if err == nil {
			t.Fatal("call-me-fail should have failed during call")
		}
	})
}

func TestEmitNotification(t *testing.T) {
	t.Run("successful-emission", func(t *testing.T) {
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
		err = EmitNotification("foo-v1", "bar",
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
	t.Run("emit-invalid-type", func(t *testing.T) {
		resetTestBus()
		err := EmitNotification("foo-v1", "bar", make(chan struct{}))
		if err == nil {
			t.Fatal("should have failed, invalid type emitted")
		}
	})
	t.Run("dial-failed", func(t *testing.T) {
		resetTestBus()
		tBus.toggleDialFailure()
		defer tBus.toggleDialFailure()

		err := EmitNotification("foo-v1", "bar",
			map[string]interface{}{
				"baz": "quux",
			})
		if err == nil {
			t.Fatal("should have failed, client can't connect")
		}
	})
}
