// Copyright (c) 2017,2019, AT&T Intellectual Property. All rights reserved.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package vci

import (
	"testing"
	"time"
)

func TestSubscription(t *testing.T) {
	t.Run("run", testRun)
	t.Run("deliver", testDeliver)
	t.Run("caching", testCaching)
	t.Run("coalescing", testCoalescing)
	t.Run("dropping", testDropping)
	t.Run("blocking", testBlocking)
	t.Run("remove-limit", testRemoveLimit)
	t.Run("cancel", testCancel)
}

func testRun(t *testing.T) {
	resetTestBus()
	client, err := Dial()
	if err != nil {
		t.Fatal(err)
	}
	t.Run("valid-subscriber", func(t *testing.T) {
		done := make(chan struct{})
		sub := client.Subscribe("foo", "bar",
			func(in map[string]interface{}) {
				close(done)
			})
		err = sub.Run()
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("invalid-subscriber", func(t *testing.T) {
		done := make(chan struct{})
		sub := client.Subscribe("foo", "bar",
			func(in map[string]interface{}, in2 int) {
				close(done)
			})
		err = sub.Run()
		if err == nil {
			t.Fatal("expected error did not occur")
		}
	})
	t.Run("run-called-twice", func(t *testing.T) {
		done := make(chan struct{})
		sub := client.Subscribe("foo", "bar",
			func(in map[string]interface{}) {
				close(done)
			})
		err = sub.Run()
		if err != nil {
			t.Fatal(err)
		}
		err = sub.Run()
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("run-called-on-broken-client", func(t *testing.T) {
		resetTestBus()
		tBus.toggleDialFailure()
		client, _ := Dial()
		done := make(chan struct{})
		sub := client.Subscribe("foo", "bar",
			func(in map[string]interface{}) {
				close(done)
			})
		err = sub.Run()
		if err == nil {
			t.Fatal("expected error did not occur")
		}
	})
}

func testDeliver(t *testing.T) {
	t.Run("deliver-to-subscriber", func(t *testing.T) {
		resetTestBus()
		client, err := Dial()
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan struct{})
		sub := client.Subscribe("foo", "bar",
			func(in map[string]interface{}) {
				close(done)
			})
		err = sub.Run()
		if err != nil {
			t.Fatal(err)
		}
		err = sub.Deliver(`{"baz":"quux"}`)
		if err != nil {
			t.Fatal(err)
		}
		select {
		case <-done:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("didn't receive expected notification")
		}

	})
	t.Run("deliver-to-string-subscriber", func(t *testing.T) {
		resetTestBus()
		client, err := Dial()
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan struct{})
		sub := client.Subscribe("foo", "bar",
			func(in string) {
				close(done)
			})
		err = sub.Run()
		if err != nil {
			t.Fatal(err)
		}
		err = sub.Deliver(`{"baz":"quux"}`)
		if err != nil {
			t.Fatal(err)
		}
		select {
		case <-done:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("didn't receive expected notification")
		}
	})
	t.Run("deliver-to-[]byte-subscriber", func(t *testing.T) {
		resetTestBus()
		client, err := Dial()
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan struct{})
		sub := client.Subscribe("foo", "bar",
			func(in []byte) {
				close(done)
			})
		err = sub.Run()
		if err != nil {
			t.Fatal(err)
		}
		err = sub.Deliver(`{"baz":"quux"}`)
		if err != nil {
			t.Fatal(err)
		}
		select {
		case <-done:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("didn't receive expected notification")
		}
	})
	t.Run("deliver-to-bogus-subscriber", func(t *testing.T) {
		resetTestBus()
		client, err := Dial()
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan struct{})
		sub := client.Subscribe("foo", "bar",
			func(in chan struct{}) {
				close(done)
			})
		err = sub.Run()
		if err != nil {
			t.Fatal(err)
		}
		err = sub.Deliver(`{"baz":"quux"}`)
		if err != nil {
			t.Fatal(err)
		}
		select {
		case <-done:
			t.Fatal("received unexpected notification")
		case <-time.After(100 * time.Millisecond):

		}
	})
}

func testCaching(t *testing.T) {
	resetTestBus()
	client, err := Dial()
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	vals := make(chan map[string]interface{})
	sub := client.Subscribe("foo", "bar",
		func(in map[string]interface{}) {
			select {
			case <-done:
			case vals <- in:
			}
		})
	err = sub.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Run("enabled", func(t *testing.T) {
		sub.ToggleCaching()
		err = sub.Deliver(`{"baz":"quux"}`)
		if err != nil {
			t.Fatal(err)
		}
		select {
		case <-vals:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("didn't receive expected notification")
		}
		var out map[string]interface{}
		err = sub.StoreLastNotificationInto(&out)
		if err != nil {
			t.Fatal(err)
		}
		if out["baz"] != "quux" {
			t.Fatal("didn't receive expected notification")
		}
	})
	t.Run("disabled", func(t *testing.T) {
		sub.ToggleCaching()
		err = sub.Deliver(`{"baz":"quux"}`)
		if err != nil {
			t.Fatal(err)
		}
		select {
		case <-vals:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("didn't receive expected notification")
		}
		var out map[string]interface{}
		err = sub.StoreLastNotificationInto(&out)
		if err == nil {
			t.Fatal("expected error did not occur")
		}
	})
	close(done)
}

func testCoalescing(t *testing.T) {
	resetTestBus()
	client, err := Dial()
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	vals := make(chan map[string]interface{})
	sub := client.Subscribe("foo", "bar",
		func(in map[string]interface{}) {
			select {
			case <-done:
			case vals <- in:
			}
		}).Coalesce()
	err = sub.Run()
	if err != nil {
		t.Fatal(err)
	}
	for _, val := range []string{"a", "b", "c", "d", "e"} {
		err = sub.Deliver(`{"baz":"` + val + `"}`)
		if err != nil {
			t.Fatal(err)
		}
	}
	select {
	case val := <-vals:
		if val["baz"] != "e" {
			t.Fatal("unexpected notification")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("didn't receive expected notification")
	}
	close(done)
}

func testDropping(t *testing.T) {
	resetTestBus()
	client, err := Dial()
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	vals := make(chan map[string]interface{})
	sub := client.Subscribe("foo", "bar",
		func(in map[string]interface{}) {
			select {
			case <-done:
			case vals <- in:
			}
		}).DropAfterLimit(2)
	err = sub.Run()
	if err != nil {
		t.Fatal(err)
	}
	for _, val := range []string{"a", "b", "c", "d", "e"} {
		err = sub.Deliver(`{"baz":"` + val + `"}`)
		if err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 3; i++ {
		select {
		case <-vals:
			if i == 2 {
				t.Fatal("unexpected notification")
			}
		case <-time.After(100 * time.Millisecond):
			if i != 2 {
				t.Fatal("didn't receive expected notification")
			}
		}
	}
	close(done)
}

func testBlocking(t *testing.T) {
	resetTestBus()
	client, err := Dial()
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	vals := make(chan map[string]interface{})
	sub := client.Subscribe("foo", "bar",
		func(in map[string]interface{}) {
			select {
			case <-done:
			case vals <- in:
			}
		}).BlockAfterLimit(2)
	err = sub.Run()
	if err != nil {
		t.Fatal(err)
	}
	errch := make(chan error)
	go func() {
		for _, val := range []string{"a", "b", "c", "d", "e"} {
			err = sub.Deliver(`{"baz":"` + val + `"}`)
			if err != nil {
				select {
				case errch <- err:
				case <-done:
				}
			}
		}
	}()
	for i := 0; i < 6; i++ {
		select {
		case <-errch:
			t.Fatal(err)
		case <-vals:
			if i == 5 {
				t.Fatal("unexpected notification")
			}
		case <-time.After(100 * time.Millisecond):
			if i != 5 {
				t.Fatal("didn't receive expected notification")
			}
		}
	}
	close(done)
}

func testRemoveLimit(t *testing.T) {
	resetTestBus()
	client, err := Dial()
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	vals := make(chan map[string]interface{})
	sub := client.Subscribe("foo", "bar",
		func(in map[string]interface{}) {
			select {
			case <-done:
			case vals <- in:
			}
		}).DropAfterLimit(2)
	err = sub.Run()
	if err != nil {
		t.Fatal(err)
	}
	sub.RemoveLimit()
	for _, val := range []string{"a", "b", "c", "d", "e"} {
		err = sub.Deliver(`{"baz":"` + val + `"}`)
		if err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 6; i++ {
		select {
		case <-vals:
			if i == 5 {
				t.Fatal("unexpected notification")
			}
		case <-time.After(100 * time.Millisecond):
			if i != 5 {
				t.Fatal("didn't receive expected notification")
			}
		}
	}
	close(done)
}

func testCancel(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		resetTestBus()
		client, err := Dial()
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan struct{})
		vals := make(chan map[string]interface{})
		sub := client.Subscribe("foo", "bar",
			func(in map[string]interface{}) {
				select {
				case <-done:
				case vals <- in:
				}
			})
		err = sub.Run()
		if err != nil {
			t.Fatal(err)
		}
		err = sub.Cancel()
		if err != nil {
			t.Fatal(err)
		}
		if !sub.isDone() {
			t.Fatal("cancel didn't stop the subscriber")
		}
		for i := 0; i < 10; i++ {
			if sub.isRunning() {
				time.Sleep(10 * time.Millisecond)
			} else {
				break
			}
			if i == 9 {
				t.Fatal("subscriber didn't shutdown")
			}
		}
		close(done)
	})
	t.Run("cancel-twice", func(t *testing.T) {
		resetTestBus()
		client, err := Dial()
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan struct{})
		vals := make(chan map[string]interface{})
		sub := client.Subscribe("foo", "bar",
			func(in map[string]interface{}) {
				select {
				case <-done:
				case vals <- in:
				}
			})
		err = sub.Run()
		if err != nil {
			t.Fatal(err)
		}
		err = sub.Cancel()
		if err != nil {
			t.Fatal(err)
		}
		for sub.isRunning() {
			time.Sleep(10 * time.Millisecond)
		}
		err = sub.Cancel()
		if err != nil {
			t.Fatal(err)
		}
	})
}
