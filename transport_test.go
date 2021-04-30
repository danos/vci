// Copyright (c) 2017,2019, 2021, AT&T Intellectual Property.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0
//

package vci

import (
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/danos/mgmterror"
	"github.com/danos/vci/internal/queue"
)

var tBus *testBus

const (
	testYangServiceName   = "net.vyatta.vci.config.yangd"
	testYangServiceModel  = "net.vyatta.vci.config.yangd.v1"
	testYangServiceModule = "yangd-v1"
)

const emptyMetadata = "{}"

func initTestBus() {
	tBus = newTestBus()
	setDefaultTransportConstructor(func() transporter {
		return newTestTransport()
	})
	ys := newTestYangService()
	comp := NewComponent(testYangServiceName)
	comp.Model(testYangServiceModel).
		RPC(testYangServiceModule, ys)
	_, err := ys.RegisterModule(map[string]interface{}{
		"name":        testYangServiceModule,
		"destination": testYangServiceModel,
	})
	if err != nil {
		panic(err)
	}
	err = comp.Run()
	if err != nil {
		panic(err)
	}
}

func resetTestBus() {
	initTestBus()
}

type testBus struct {
	mu              sync.Mutex
	failDial        bool
	connections     []*testConn
	connectionsByID map[string]*testConn
	subscriptions   map[string][]transportSubscriber
}

func newTestBus() *testBus {
	return &testBus{
		connections:     make([]*testConn, 0),
		connectionsByID: make(map[string]*testConn),
		subscriptions:   make(map[string][]transportSubscriber),
	}
}

func (b *testBus) toggleDialFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failDial = !b.failDial
}

func (b *testBus) removeConnection(conn *testConn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	new := make([]*testConn, 0, len(b.connections))
	for _, c := range b.connections {
		if c == conn {
			continue
		}
		new = append(new, c)
	}
	ids := make(map[string]struct{})
	for id, c := range b.connectionsByID {
		if c == conn {
			ids[id] = struct{}{}
		}
	}
	for id := range ids {
		delete(b.connectionsByID, id)
	}
	b.connections = new
}

func (b *testBus) Dial() *testConn {
	b.mu.Lock()
	defer b.mu.Unlock()
	c := newTestConn(b, b.failDial)
	b.connections = append(b.connections, c)
	return c
}

func (b *testBus) Object(id, objectName string) (*testObject, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	conn, ok := b.connectionsByID[id]
	if !ok {
		return nil, errors.New("unknown object")
	}
	return conn.connectedObject(objectName)
}

func (b *testBus) requestID(id string, c *testConn) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.connectionsByID[id]; ok {
		return errors.New("id already in use")
	}
	b.connectionsByID[id] = c
	return nil
}

func (b *testBus) Subscribe(notificationName string, s transportSubscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()
	subs := b.subscriptions[notificationName]
	for _, sub := range subs {
		if sub == s {
			return
		}
	}
	b.subscriptions[notificationName] =
		append(b.subscriptions[notificationName], s)
}

func (b *testBus) Unsubscribe(notificationName string, s transportSubscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()
	subs := b.subscriptions[notificationName]
	newSubs := make([]transportSubscriber, 0, len(subs))
	for _, sub := range subs {
		if sub == s {
			continue
		}
		newSubs = append(newSubs, sub)
	}
	if len(newSubs) == 0 {
		delete(b.subscriptions, notificationName)
	}
	b.subscriptions[notificationName] = newSubs
}

func (b *testBus) UnsubscribeAll(notificationName string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.subscriptions, notificationName)
}

func (b *testBus) Emit(notificationName string, input string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	subs, ok := b.subscriptions[notificationName]
	if !ok {
		return
	}
	for _, sub := range subs {
		_ = sub.Deliver(input)
	}
}

type testConn struct {
	bus     *testBus
	id      string
	failed  bool
	objects map[string]*testObject
}

func newTestConn(bus *testBus, failed bool) *testConn {
	return &testConn{
		bus:     bus,
		failed:  failed,
		objects: make(map[string]*testObject),
	}
}

func (c *testConn) testConnection() error {
	if c.failed {
		return errors.New("client is not connected")
	}
	return nil
}

func (c *testConn) Object(id, objectName string) (*testObject, error) {
	err := c.testConnection()
	if err != nil {
		return nil, err
	}
	return c.bus.Object(id, objectName)
}

func (c *testConn) connectedObject(name string) (*testObject, error) {
	obj, ok := c.objects[name]
	if !ok {
		return nil, errors.New("unknown object")
	}
	return obj, nil
}

func (c *testConn) RequestIdentity(id string) error {
	err := c.testConnection()
	if err != nil {
		return err
	}
	err = c.bus.requestID(id, c)
	if err != nil {
		return err
	}
	c.id = id
	for name, object := range c.objects {
		if object.Type() != "rpc" || name == testYangServiceModule {
			continue
		}
		//Since we don't have Yangd or component descriptors in the
		//test environment so we have to fake the registration.
		var out string
		obj, err := c.Object(testYangServiceModel, testYangServiceModule)
		if err != nil {
			return err
		}
		call, err := obj.Call("register-module",
			emptyMetadata,
			"{\"name\":\""+name+"\","+
				"\"destination\":\""+c.id+"\"}")
		if err != nil {
			return err
		}
		err = call.StoreOutputInto(&out)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *testConn) Export(object transportObject) error {
	err := c.testConnection()
	if err != nil {
		return err
	}
	_, ok := c.objects[object.Name()]
	if ok {
		return errors.New("requested object already exists")
	}
	c.objects[object.Name()] = &testObject{
		methods: object.Methods(),
		typ:     object.Type(),
	}
	return nil
}

func (c *testConn) Subscribe(notificationName string, sub transportSubscriber) error {
	err := c.testConnection()
	if err != nil {
		return err
	}
	c.bus.Subscribe(notificationName, sub)
	return nil
}

func (c *testConn) Unsubscribe(notificationName string, sub transportSubscriber) error {
	err := c.testConnection()
	if err != nil {
		return err
	}
	c.bus.Unsubscribe(notificationName, sub)
	return nil
}

func (c *testConn) UnsubscribeAll(notificationName string) error {
	err := c.testConnection()
	if err != nil {
		return err
	}
	c.bus.UnsubscribeAll(notificationName)
	return nil
}

func (c *testConn) Emit(notificationName string, input string) error {
	err := c.testConnection()
	if err != nil {
		return err
	}
	c.bus.Emit(notificationName, input)
	return nil
}

func (c *testConn) Close() error {
	c.bus.removeConnection(c)
	return nil
}

type testObject struct {
	typ     string
	methods map[string]interface{}
}

func (o *testObject) Type() string {
	return o.typ
}

func (o *testObject) Call(name string, meta, encodedData string) (*testRPCPromise, error) {
	//good enough to test our APIs but not completely generic.
	if o == nil {
		return nil, errors.New("Unknown object")
	}
	method, ok := o.methods[name]
	if !ok {
		return nil, errors.New("Unknown method")
	}
	mVal := reflect.ValueOf(method)
	metaVal := reflect.ValueOf(meta)
	inVal := reflect.ValueOf(encodedData)
	var outVals []reflect.Value
	switch mVal.Type().NumIn() {
	case 0:
		outVals = mVal.Call([]reflect.Value{})
	case 1:
		outVals = mVal.Call([]reflect.Value{inVal})
	case 2:
		outVals = mVal.Call([]reflect.Value{metaVal, inVal})
	}

	switch mVal.Type().NumOut() {
	case 0:
		return &testRPCPromise{}, nil
	case 1:
		if mVal.Type().Out(0) == reflect.TypeOf(error(nil)) {
			return &testRPCPromise{
				err: outVals[0].Interface().(error),
			}, nil
		}
		return &testRPCPromise{
			out: "",
		}, nil
	case 2:
		if outVals[1].Interface() != nil {
			return &testRPCPromise{
				err: outVals[1].Interface().(error),
			}, nil
		}
	}
	return &testRPCPromise{
		out: outVals[0].Interface().(string),
	}, nil
}

type testRPCPromise struct {
	err error
	out string
}

func (p *testRPCPromise) StoreOutputInto(out *string) error {
	if p.err != nil {
		return p.err
	}
	*out = p.out
	return nil
}

type testSubscriber struct {
	queue queue.Queue
}

func newTestSubscriber(queue queue.Queue) *testSubscriber {
	return &testSubscriber{
		queue: queue,
	}
}
func (s *testSubscriber) Deliver(in string) error {
	s.queue.Enqueue(in)
	return nil
}

type testTransport struct {
	conn *testConn
}

func newTestTransport() *testTransport {
	return &testTransport{}
}

func (t *testTransport) Dial() error {
	t.conn = tBus.Dial()
	if t.conn.failed {
		return errors.New("failed to connect to bus")
	}
	return nil
}

func (t *testTransport) RequestIdentity(id string) error {
	return t.conn.RequestIdentity(id)
}

func (t *testTransport) Call(moduleName, rpcName, meta, input string) (transportRPCPromise, error) {
	modelName, err := t.getDestinationByModuleName(moduleName)
	if err != nil {
		return nil, err
	}
	obj, err := t.conn.Object(modelName, moduleName)
	if err != nil {
		return nil, err
	}
	return obj.Call(rpcName, meta, input)
}
func (t *testTransport) Subscribe(
	moduleName, notificationName string,
	subscriber transportSubscriber,
) error {
	name := moduleName + "/" + notificationName
	return t.conn.Subscribe(name, subscriber)
}
func (t *testTransport) Unsubscribe(
	moduleName, notificationName string,
	subscriber transportSubscriber,
) error {
	name := moduleName + "/" + notificationName
	//TODO: this interface should pass in the queue so
	//there can be multiple subscriptions per name.
	return t.conn.Unsubscribe(name, subscriber)
}
func (t *testTransport) Emit(moduleName, notificationName, encodedData string) error {
	name := moduleName + "/" + notificationName
	return t.conn.Emit(name, encodedData)
}
func (t *testTransport) SetConfigForModel(
	modelName string, encodedData string) error {
	obj, err := t.conn.Object(modelName, "running")
	if err != nil {
		return err
	}
	_, err = obj.Call("set", emptyMetadata, encodedData)
	return err
}
func (t *testTransport) CheckConfigForModel(
	modelName string, encodedData string) error {
	obj, err := t.conn.Object(modelName, "running")
	if err != nil {
		return err
	}
	_, err = obj.Call("check", emptyMetadata, encodedData)
	return err
}
func (t *testTransport) StoreConfigByModelInto(modelName string, encodedData *string) error {
	obj, err := t.conn.Object(modelName, "running")
	if err != nil {
		return err
	}
	call, err := obj.Call("get", emptyMetadata, "")
	if err != nil {
		return err
	}
	return call.StoreOutputInto(encodedData)
}
func (t *testTransport) StoreStateByModelInto(modelName string, encodedData *string) error {
	obj, err := t.conn.Object(modelName, "state")
	if err != nil {
		return err
	}
	call, err := obj.Call("get", emptyMetadata, "")
	if err != nil {
		return err
	}
	return call.StoreOutputInto(encodedData)
}
func (t *testTransport) Export(object transportObject) error {
	if !object.IsValid() {
		if err, ok := object.(error); ok {
			return err
		}
		return errors.New("invalid object")
	}
	return t.conn.Export(object)
}

func (t *testTransport) Close() error {
	return t.conn.Close()
}

func (t *testTransport) getDestinationByModuleName(
	moduleName string,
) (string, error) {
	var data string
	var out string
	var err error

	marshaller := defaultMarshaller()
	in := map[string]interface{}{
		yangdModuleName + ":module-name": moduleName,
	}

	if data, err = marshaller.Marshal(in); err != nil {
		return "", mgmterror.NewMalformedMessageError()
	}
	obj, err := t.conn.Object(testYangServiceModel, testYangServiceModule)
	if err != nil {
		return "", err
	}
	call, err := obj.Call("lookup-rpc-destination-by-module-name", emptyMetadata, data)
	if err != nil {
		return "", err
	}
	err = call.StoreOutputInto(&out)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err = marshaller.Unmarshal(out, &result); err != nil {
		return "", mgmterror.NewMalformedMessageError()
	}

	return result[yangdModuleName+":destination"].(string), nil
}

type testYangService struct {
	mapping map[string]string
}

func newTestYangService() *testYangService {
	return &testYangService{
		mapping: make(map[string]string),
	}
}

func (ys *testYangService) RegisterModule(
	in map[string]interface{},
) (struct{}, error) {
	ys.mapping[in["name"].(string)] = in["destination"].(string)
	return struct{}{}, nil
}

func (ys *testYangService) LookupRpcDestinationByModuleName(
	encodedData string,
) (map[string]interface{}, error) {
	//Note: This methods must bypass the marshalling
	//      it is used internally in the wrappers
	//      as a resource for other components.
	//      Any real implementation of Yangd will need
	//      to do this as well.
	marshaller := defaultMarshaller()
	var in map[string]interface{}
	if err := marshaller.Unmarshal(encodedData, &in); err != nil {
		return nil, err
	}
	modelName, ok := ys.mapping[in[yangdModuleName+":module-name"].(string)]
	if !ok {
		return nil, errors.New("unknown module name")
	}
	return map[string]interface{}{
		yangdModuleName + ":destination": modelName,
	}, nil
}

func (ys *testYangService) ValidateRpcInput(
	encodedData string,
) (map[string]interface{}, error) {
	//Note: This methods must bypass the marshalling
	//      it is used internally in the wrappers
	//      as a resource for other components.
	//      Any real implementation of Yangd will need
	//      to do this as well.
	marshaller := defaultMarshaller()
	var in map[string]interface{}
	if err := marshaller.Unmarshal(encodedData, &in); err != nil {
		return map[string]interface{}{
			yangdModuleName + ":valid": false,
		}, err
	}

	var rpcInput map[string]interface{}
	if err := marshaller.Unmarshal(in[yangdModuleName+":rpc-input"].(string),
		&rpcInput); err != nil {
		return map[string]interface{}{
			yangdModuleName + ":valid": false,
		}, err
	}
	return map[string]interface{}{
		yangdModuleName + ":valid": true,
	}, nil
}

func (ys *testYangService) ValidateNotification(
	in map[string]string,
) (map[string]string, error) {
	out := make(map[string]string)
	out[yangdModuleName+":output"] = in[yangdModuleName+":input"]
	return out, nil
}

func setupTestYangService(modules map[string]string) error {
	ys := newTestYangService()
	comp := NewComponent(testYangServiceName)
	comp.Model(testYangServiceModel).
		RPC(testYangServiceModule, ys)
	err := comp.Run()
	if err != nil {
		return err
	}
	_, _ = ys.RegisterModule(map[string]interface{}{
		"name":        testYangServiceModule,
		"destination": testYangServiceModel,
	})
	for module, model := range modules {
		_, _ = ys.RegisterModule(map[string]interface{}{
			"name":        module,
			"destination": model,
		})
	}
	return nil
}

func testTransportSemantics(t *testing.T, transport transporter) {
	//Since there is no easy way to reset the bus between these
	//calls, a failure may cause a cascade through all the below
	//cases.
	testModule := "test-v1"
	testModel := "net.vyatta.test"
	err := setupTestYangService(map[string]string{
		testModule: testModel,
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("DialSucceeds", func(t *testing.T) {
		//Dial() error
		err := transport.Dial()
		if err != nil {
			t.Fatal(err)
		}
		err = transport.Close()
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("RequestIdentitySucceeds", func(t *testing.T) {
		//RequestIdentity(id string) error
		err := transport.Dial()
		if err != nil {
			t.Fatal(err)
		}
		err = transport.RequestIdentity(testModel)
		if err != nil {
			t.Fatal(err)
		}
		err = transport.Close()
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Export", func(t *testing.T) {
		//Export(object transportObject) error
		err := transport.Dial()
		if err != nil {
			t.Fatal(err)
		}
		err = transport.RequestIdentity(testModel)
		if err != nil {
			t.Fatal(err)
		}
		t.Run("config", func(t *testing.T) {
			err := transport.Export(newConfig(&testRunningConfig{},
				newClient().withTransport(transport)))
			if err != nil {
				t.Fatal(err)
			}
		})
		t.Run("state", func(t *testing.T) {
			err := transport.Export(newState(&testState{},
				newClient().withTransport(transport)))
			if err != nil {
				t.Fatal(err)
			}
		})
		t.Run("RPC", func(t *testing.T) {
			err := transport.Export(newRPC(testModule,
				map[string]interface{}{
					"foo": func(in string) (string, error) {
						return in, nil
					},
					"fail": func(in string) (string, error) {
						return "", errors.New("fail")
					},
				},
				newClient().withTransport(transport)))
			if err != nil {
				t.Fatal(err)
			}
		})
		t.Run("invalid", func(t *testing.T) {
			err := transport.Export(newRPC("foo", nil, nil))
			if err == nil {
				t.Fatal("expected error did not occur")
			}
		})
		err = transport.Close()
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Call", func(t *testing.T) {
		//Call(moduleName, rpcName, input string) (transportRPCPromise, error)
		err := transport.Dial()
		if err != nil {
			t.Fatal(err)
		}
		err = transport.RequestIdentity(testModel)
		if err != nil {
			t.Fatal(err)
		}
		err = transport.Export(newRPC(testModule,
			map[string]interface{}{
				"foo": func(in string) (string, error) {
					return in, nil
				},
				"fail": func(in string) (string, error) {
					return "", errors.New("fail")
				},
			},
			newClient().withTransport(transport)))
		if err != nil {
			t.Fatal(err)
		}
		in := "{\"value\":\"bar\"}"
		t.Run("success", func(t *testing.T) {
			var out string

			call, err := transport.Call("test-v1", "foo", emptyMetadata, in)
			if err != nil {
				t.Fatal(err)
			}

			err = call.StoreOutputInto(&out)
			if err != nil {
				t.Fatal(err)
			}

			if out != in {
				t.Fatal("foo didn't return the right data")
			}
		})
		t.Run("fail", func(t *testing.T) {
			var out string
			call, err := transport.Call("test-v1", "fail", emptyMetadata, in)
			if err != nil {
				t.Fatal(err)
			}

			err = call.StoreOutputInto(&out)
			if err == nil {
				t.Fatal("expected error didn't occur")
			}
		})
		t.Run("non-existant-module", func(t *testing.T) {
			_, err := transport.Call("test-v2", "foo", emptyMetadata, in)
			if err == nil {
				t.Fatal("expected failure didn't occur")
			}

		})
		t.Run("non-existant-rpc", func(t *testing.T) {
			_, err := transport.Call("test-v1", "bar", emptyMetadata, in)
			if err == nil {
				t.Fatal("expected failure didn't occur")
			}

		})
		err = transport.Close()
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("StoreConfigByModelInto", func(t *testing.T) {
		//StoreConfigByModelInto(modelName string, encodedData *string) error
		err := transport.Dial()
		if err != nil {
			t.Fatal(err)
		}
		err = transport.RequestIdentity(testModel)
		if err != nil {
			t.Fatal(err)
		}
		t.Run("call-when-no-config-is-exposed", func(t *testing.T) {
			var out string
			err := transport.Export(newState(
				&testState{Value: "foo bar"},
				newClient().withTransport(transport)))
			if err != nil {
				t.Fatal(err)
			}
			err = transport.StoreConfigByModelInto(testModel, &out)
			if err == nil {
				t.Fatal("expected failure didn't occur")
			}
		})
		t.Run("successful-call", func(t *testing.T) {
			var out string
			exp := `{"value":"foo bar"}`
			err := transport.Export(newConfig(&testRunningConfig{},
				newClient().withTransport(transport)))
			if err != nil {
				t.Fatal(err)
			}
			err = transport.StoreConfigByModelInto(testModel, &out)
			if err != nil {
				t.Fatal(err)
			}
			if out != exp {
				t.Fatalf("expected %q, got %q",
					exp,
					out)
			}

		})
		err = transport.Close()
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("StoreStateByModelInto", func(t *testing.T) {
		//StoreStateByModelInto(modelName string, encodedData *string) error
		err := transport.Dial()
		if err != nil {
			t.Fatal(err)
		}
		err = transport.RequestIdentity(testModel)
		if err != nil {
			t.Fatal(err)
		}
		t.Run("call-when-no-state-is-exposed", func(t *testing.T) {
			var out string
			err := transport.Export(newConfig(&testRunningConfig{},
				newClient().withTransport(transport)))
			if err != nil {
				t.Fatal(err)
			}
			err = transport.StoreStateByModelInto(testModel, &out)
			if err == nil {
				t.Fatal("expected failure didn't occur")
			}
		})
		t.Run("successful-call", func(t *testing.T) {
			var out string
			exp := `{"value":"foo bar"}`
			err := transport.Export(newState(
				&testState{Value: "foo bar"},
				newClient().withTransport(transport)))
			if err != nil {
				t.Fatal(err)
			}
			err = transport.StoreStateByModelInto(testModel, &out)
			if err != nil {
				t.Fatal(err)
			}
			if out != exp {
				t.Fatalf("expected %q, got %q",
					exp,
					out)
			}

		})
		err = transport.Close()
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("EmitSucceeds", func(t *testing.T) {
		//Emit(moduleName, notificationName, encodedData string) error
		err := transport.Dial()
		if err != nil {
			t.Fatal(err)
		}
		err = transport.RequestIdentity(testModel)
		if err != nil {
			t.Fatal(err)
		}
		err = transport.Emit("foo-v1", "bar", `{"baz":"quux"}`)
		if err != nil {
			t.Fatal(err)
		}
		err = transport.Close()
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Subscribe", func(t *testing.T) {
		//Subscribe(moduleName, notificationName string, queue queue.Queue) error
		t.Run("normal", func(t *testing.T) {
			err := transport.Dial()
			if err != nil {
				t.Fatal(err)
			}
			err = transport.RequestIdentity(testModel)
			if err != nil {
				t.Fatal(err)
			}
			notif := `{"baz":"quux"}`
			sub := newTestSubscriber(queue.NewUnbounded())
			vals := make(chan interface{})
			done := make(chan struct{})
			go func() {
				var finished bool
				for !finished {
					select {
					case vals <- sub.queue.Dequeue():
					case <-done:
						finished = true
					}
				}
			}()
			err = transport.Subscribe("foo-v1", "bar", sub)
			if err != nil {
				t.Fatal(err)
			}
			err = transport.Emit("foo-v1", "bar", notif)
			if err != nil {
				t.Fatal(err)
			}
			select {
			case val := <-vals:
				if val != notif {
					t.Fatalf("expected %q, got %q",
						notif, val)
				}
			case <-time.After(100 * time.Millisecond):
				t.Fatal("didn't receive expected notification")
			}
			close(done)
			err = transport.Close()
			if err != nil {
				t.Fatal(err)
			}
		})
		t.Run("deduped-subscribers", func(t *testing.T) {
			err := transport.Dial()
			if err != nil {
				t.Fatal(err)
			}
			err = transport.RequestIdentity(testModel)
			if err != nil {
				t.Fatal(err)
			}
			notif := `{"baz":"quux"}`
			sub := newTestSubscriber(queue.NewUnbounded())
			vals := make(chan interface{})
			done := make(chan struct{})
			go func() {
				var finished bool
				for !finished {
					select {
					case vals <- sub.queue.Dequeue():
					case <-done:
						finished = true
					}
				}
			}()
			err = transport.Subscribe("foo-v1", "bar", sub)
			if err != nil {
				t.Fatal(err)
			}
			err = transport.Subscribe("foo-v1", "bar", sub)
			if err != nil {
				t.Fatal(err)
			}
			err = transport.Emit("foo-v1", "bar", notif)
			if err != nil {
				t.Fatal(err)
			}
			select {
			case val := <-vals:
				if val != notif {
					t.Fatalf("expected %q, got %q",
						notif, val)
				}
			case <-time.After(100 * time.Millisecond):
				t.Fatal("didn't receive expected notification")
			}
			select {
			case <-vals:
				t.Fatal("unexpected notification")
			case <-time.After(100 * time.Millisecond):
			}
			close(done)
			err = transport.Close()
			if err != nil {
				t.Fatal(err)
			}
		})
		t.Run("multiple-subscribers", func(t *testing.T) {
			err := transport.Dial()
			if err != nil {
				t.Fatal(err)
			}
			err = transport.RequestIdentity(testModel)
			if err != nil {
				t.Fatal(err)
			}
			notif := `{"baz":"quux"}`
			sub := newTestSubscriber(queue.NewUnbounded())
			sub2 := newTestSubscriber(queue.NewUnbounded())

			vals := make(chan interface{})
			vals2 := make(chan interface{})
			done := make(chan struct{})
			var subfn = func(sub *testSubscriber,
				vals chan interface{}) {
				var finished bool
				for !finished {
					select {
					case vals <- sub.queue.Dequeue():
					case <-done:
						finished = true
					}
				}
			}
			go subfn(sub, vals)
			go subfn(sub2, vals2)
			err = transport.Subscribe("foo-v1", "bar", sub)
			if err != nil {
				t.Fatal(err)
			}
			err = transport.Subscribe("foo-v1", "bar", sub2)
			if err != nil {
				t.Fatal(err)
			}
			err = transport.Emit("foo-v1", "bar", notif)
			if err != nil {
				t.Fatal(err)
			}
			select {
			case val := <-vals:
				if val != notif {
					t.Fatalf("expected %q, got %q",
						notif, val)
				}
			case <-time.After(100 * time.Millisecond):
				t.Fatal("didn't receive expected notification")
			}
			select {
			case val := <-vals2:
				if val != notif {
					t.Fatalf("expected %q, got %q",
						notif, val)
				}
			case <-time.After(100 * time.Millisecond):
				t.Fatal("didn't receive expected notification")
			}

			close(done)
			err = transport.Close()
			if err != nil {
				t.Fatal(err)
			}
		})
	})
	t.Run("Unsubscribe", func(t *testing.T) {
		//Unsubscribe(moduleName, notificationName string) error
		t.Run("single-unsubscribe", func(t *testing.T) {
			err := transport.Dial()
			if err != nil {
				t.Fatal(err)
			}
			err = transport.RequestIdentity(testModel)
			if err != nil {
				t.Fatal(err)
			}
			notif := `{"baz":"quux"}`
			sub := newTestSubscriber(queue.NewUnbounded())
			vals := make(chan interface{})
			done := make(chan struct{})
			go func() {
				var finished bool
				for !finished {
					select {
					case vals <- sub.queue.Dequeue():
					case <-done:
						finished = true
					}
				}
			}()
			err = transport.Subscribe("foo-v1", "bar", sub)
			if err != nil {
				t.Fatal(err)
			}
			err = transport.Emit("foo-v1", "bar", notif)
			if err != nil {
				t.Fatal(err)
			}
			select {
			case val := <-vals:
				if val != notif {
					t.Fatalf("expected %q, got %q", notif, val)
				}
			case <-time.After(100 * time.Millisecond):
				t.Fatal("didn't receive expected notification")
			}

			err = transport.Unsubscribe("foo-v1", "bar", sub)
			if err != nil {
				t.Fatal(err)
			}
			err = transport.Emit("foo-v1", "bar", notif)
			if err != nil {
				t.Fatal(err)
			}
			select {
			case <-vals:
				t.Fatal("receiveived unexpected notification")
			case <-time.After(100 * time.Millisecond):

			}
			close(done)
			err = transport.Close()
			if err != nil {
				t.Fatal(err)
			}
		})
		t.Run("multiple-subscribers-single-unsubscribe", func(t *testing.T) {
			err := transport.Dial()
			if err != nil {
				t.Fatal(err)
			}
			err = transport.RequestIdentity(testModel)
			if err != nil {
				t.Fatal(err)
			}
			notif := `{"baz":"quux"}`
			sub := newTestSubscriber(queue.NewUnbounded())
			sub2 := newTestSubscriber(queue.NewUnbounded())

			vals := make(chan interface{})
			vals2 := make(chan interface{})
			done := make(chan struct{})
			var subfn = func(sub *testSubscriber,
				vals chan interface{}) {
				var finished bool
				for !finished {
					select {
					case vals <- sub.queue.Dequeue():
					case <-done:
						finished = true
					}
				}
			}
			go subfn(sub, vals)
			go subfn(sub2, vals2)
			err = transport.Subscribe("foo-v1", "bar", sub)
			if err != nil {
				t.Fatal(err)
			}
			err = transport.Subscribe("foo-v1", "bar", sub2)
			if err != nil {
				t.Fatal(err)
			}
			err = transport.Unsubscribe("foo-v1", "bar", sub2)
			if err != nil {
				t.Fatal(err)
			}
			err = transport.Emit("foo-v1", "bar", notif)
			if err != nil {
				t.Fatal(err)
			}
			select {
			case val := <-vals:
				if val != notif {
					t.Fatalf("expected %q, got %q",
						notif, val)
				}
			case <-time.After(100 * time.Millisecond):
				t.Fatal("didn't receive expected notification")
			}
			select {
			case <-vals2:
				t.Fatal("unexpected notification")
			case <-time.After(100 * time.Millisecond):
			}

			close(done)
			err = transport.Close()
			if err != nil {
				t.Fatal(err)
			}
		})
		t.Run("multiple-subscribers-sll-unsubscribe", func(t *testing.T) {
			err := transport.Dial()
			if err != nil {
				t.Fatal(err)
			}
			err = transport.RequestIdentity(testModel)
			if err != nil {
				t.Fatal(err)
			}
			notif := `{"baz":"quux"}`
			sub := newTestSubscriber(queue.NewUnbounded())
			sub2 := newTestSubscriber(queue.NewUnbounded())

			vals := make(chan interface{})
			vals2 := make(chan interface{})
			done := make(chan struct{})
			var subfn = func(sub *testSubscriber,
				vals chan interface{}) {
				var finished bool
				for !finished {
					select {
					case vals <- sub.queue.Dequeue():
					case <-done:
						finished = true
					}
				}
			}
			go subfn(sub, vals)
			go subfn(sub2, vals2)
			err = transport.Subscribe("foo-v1", "bar", sub)
			if err != nil {
				t.Fatal(err)
			}
			err = transport.Subscribe("foo-v1", "bar", sub2)
			if err != nil {
				t.Fatal(err)
			}
			err = transport.Unsubscribe("foo-v1", "bar", sub)
			if err != nil {
				t.Fatal(err)
			}
			err = transport.Unsubscribe("foo-v1", "bar", sub2)
			if err != nil {
				t.Fatal(err)
			}
			err = transport.Emit("foo-v1", "bar", notif)
			if err != nil {
				t.Fatal(err)
			}
			select {
			case <-vals:
				t.Fatal("unexpected notification")
			case <-time.After(100 * time.Millisecond):
			}
			select {
			case <-vals2:
				t.Fatal("unexpected notification")
			case <-time.After(100 * time.Millisecond):
			}

			close(done)
			err = transport.Close()
			if err != nil {
				t.Fatal(err)
			}
		})
	})

}

func TestMockTransportSemantics(t *testing.T) {
	tBus = newTestBus()
	setDefaultTransportConstructor(func() transporter {
		return newTestTransport()
	})
	//Make sure changes to the test framework don't violate
	//the transport constraints.
	testTransportSemantics(t, newTestTransport())
}
