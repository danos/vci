// Copyright (c) 2017-2019, 2021, AT&T Intellectual Property.
// All rights reserved.
//
// Copyright (c) 2016 by Brocade Communications Systems, Inc.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package vci

import (
	"bytes"
	"errors"
	"strings"
	"sync"
	"unicode"

	"github.com/coreos/go-systemd/daemon"
	"github.com/danos/mgmterror"
	"github.com/godbus/dbus"
	"github.com/godbus/dbus/introspect"
	"github.com/jsouthworth/objtree"
)

const (
	fdtDBusName        = "org.freedesktop.DBus"
	fdtAddMatch        = fdtDBusName + ".AddMatch"
	fdtRemoveMatch     = fdtDBusName + ".RemoveMatch"
	yangModuleDBusPfx  = "yang.module"
	yangdRPCPath       = "/yangd_v1/rpc"
	readDBusInterface  = "net.vyatta.vci.config.read"
	writeDBusInterface = "net.vyatta.vci.config.write"
	vciBusAddress      = "unix:path=/var/run/vci/vci_bus_socket"
)

func init() {
	// Assign the DBus transport constructor as the default
	// if porting to a new transport, removing this init
	// and using a similar function in the new transport
	// implementation will cause it to be used instead.
	setDefaultTransportConstructor(func() transporter {
		return newDBusVciTransport()
	})
}

type dbusServiceRead interface {
	Get() (string, error)
}

type dbusServiceWrite interface {
	Check(string) error
	Set(string) error
}

type dbusCall struct {
	call      *dbus.Call
	transport *dbusTransport
}

func (c *dbusCall) StoreOutputInto(output *string) error {
	call := <-c.call.Done
	err := call.Store(output)
	if err != nil {
		err = c.transport.processError(err)
	}
	return err
}

type dBusConnector func(dbus.Handler, dbus.SignalHandler) (*dbus.Conn, error)

type dbusTransport struct {
	busMgr         *objtree.BusManager
	conn           *dbus.Conn
	connectFn      dBusConnector
	signalHandlers struct {
		mu       sync.RWMutex
		handlers map[string][]transportSubscriber
	}
}

func newDBusTransport() *dbusTransport {
	t := &dbusTransport{}
	t.signalHandlers.handlers = make(map[string][]transportSubscriber)
	t.connectFn = dBusConnector(t.systemConnectFn)
	return t
}

func newDBusVciTransport() *dbusTransport {
	t := &dbusTransport{}
	t.signalHandlers.handlers = make(map[string][]transportSubscriber)
	t.connectFn = dBusConnector(t.vciConnectFn)
	return t
}

func newDBusSessionTransport() *dbusTransport {
	t := &dbusTransport{}
	t.signalHandlers.handlers = make(map[string][]transportSubscriber)
	t.connectFn = dBusConnector(t.sessionConnectFn)
	return t
}

func (t *dbusTransport) systemConnectFn(
	hdlr dbus.Handler,
	_ dbus.SignalHandler,
) (*dbus.Conn, error) {
	// This bit is a little tricky. We don't want to use the
	// objtree notification handlers as they are limiting,
	// instead we want to intercept the notifications. We can do this
	// by using the passed in handler from objtree for normal
	// calls and our own handler for signals.
	return dbus.SystemBusPrivateHandler(hdlr, t)
}

func (t *dbusTransport) vciConnectFn(
	hdlr dbus.Handler,
	_ dbus.SignalHandler,
) (*dbus.Conn, error) {
	// This bit is a little tricky. We don't want to use the
	// objtree notification handlers as they are limiting,
	// instead we want to intercept the notifications. We can do this
	// by using the passed in handler from objtree for normal
	// calls and our own handler for signals.
	return dbus.DialHandler(vciBusAddress, hdlr, t)
}

// sessionConnectFn is used for testing but we may switch to it
// for the default implementation in the future. For testing the
// system buses tend to be locked down by default and require special
// configuration to get working using the session bus to test this code
// causes no functional difference on the DBus semantics so it should
// be ok.
func (t *dbusTransport) sessionConnectFn(
	hdlr dbus.Handler,
	_ dbus.SignalHandler,
) (*dbus.Conn, error) {
	return dbus.SessionBusPrivateHandler(hdlr, t)
}

func (t *dbusTransport) DeliverSignal(
	iface, name string,
	signal *dbus.Signal,
) {
	t.signalHandlers.mu.RLock()
	defer t.signalHandlers.mu.RUnlock()

	sigName := iface + "/" + name
	subs, ok := t.signalHandlers.handlers[sigName]
	if !ok {
		return
	}
	for _, sub := range subs {
		_ = sub.Deliver(signal.Body[0].(string))
	}
}

func (t *dbusTransport) Dial() error {
	if t.conn != nil {
		return nil
	}
	busMgr, err := objtree.NewAnonymousBusManager(t.connectFn)
	if err != nil {
		return err
	}
	t.busMgr = busMgr
	t.conn = busMgr.Conn()
	return nil
}

func (t *dbusTransport) RequestIdentity(id string) error {
	_, err := daemon.SdNotify(false, "READY=1")
	if err != nil {
		return err
	}
	return t.busMgr.RequestName(id)
}

func (t *dbusTransport) Call(
	moduleName, rpcName string,
	encodedData string,
) (transportRPCPromise, error) {
	modelName, err := t.getDestinationByModuleName(moduleName)
	if err != nil {
		return nil, errors.New(
			"unable to locate RPC on Bus (no model): " +
				moduleName + ":" + rpcName)
	}
	dbusRPCName := t.convertYangNameToDBus(rpcName)
	if !t.isDBusRPC(modelName, moduleName, dbusRPCName) {
		return nil, errors.New(
			"unable to locate RPC on Bus: " +
				modelName + ":" + moduleName + ":" + rpcName)
	}

	obj := t.conn.Object(modelName, t.getModuleRPCObjectPath(moduleName))
	call := obj.Go(t.getModuleRPCInterfaceName(moduleName)+
		"."+dbusRPCName, 0, nil, encodedData)
	return &dbusCall{call: call, transport: t}, nil
}

func (t *dbusTransport) Subscribe(
	moduleName, notificationName string,
	subscriber transportSubscriber,
) error {
	sigName := t.convertYangNameToDBus(notificationName)
	ifaceName := t.getModuleNotificationInterfaceName(moduleName)
	call := t.conn.BusObject().Call(fdtAddMatch, 0,
		"type='signal',interface='"+ifaceName+"',member='"+sigName+"'")
	if call.Err != nil {
		return call.Err
	}

	name := ifaceName + "/" + sigName
	t.addSubscriber(name, subscriber)
	return nil
}

func (t *dbusTransport) Unsubscribe(
	moduleName, notificationName string,
	subscriber transportSubscriber,
) error {
	sigName := t.convertYangNameToDBus(notificationName)
	ifaceName := t.getModuleNotificationInterfaceName(moduleName)
	name := ifaceName + "/" + sigName
	numLeft := t.removeSubscriber(name, subscriber)
	if numLeft != 0 {
		return nil
	}

	call := t.conn.BusObject().Call(fdtRemoveMatch, 0,
		"type='signal',interface='"+ifaceName+"',member='"+sigName+"'")
	if call.Err != nil {
		return call.Err
	}

	return nil
}

func (t *dbusTransport) Emit(
	moduleName, name, encodedData string,
) error {
	modulePath := t.getModuleNotificationObjectPath(moduleName)
	notificationName := t.getModuleNotificationInterfaceName(moduleName) +
		"." + t.convertYangNameToDBus(name)
	return t.conn.Emit(modulePath, notificationName, encodedData)
}

func (t *dbusTransport) SetConfigForModel(
	modelName string, encodedData string,
) error {
	obj := t.conn.Object(modelName, "/running")
	err := obj.Call(writeDBusInterface+".Set", 0, encodedData).Store()
	if err != nil {
		err = t.processErrorIgnoreUnsupported(err)
	}
	return err
}

func (t *dbusTransport) CheckConfigForModel(
	modelName string, encodedData string,
) error {
	obj := t.conn.Object(modelName, "/running")
	err := obj.Call(writeDBusInterface+".Check", 0, encodedData).Store()
	if err != nil {
		err = t.processErrorIgnoreUnsupported(err)
	}
	return err
}

func (t *dbusTransport) StoreConfigByModelInto(
	modelName string, encodedData *string,
) error {
	obj := t.conn.Object(modelName, "/running")
	err := obj.Call(readDBusInterface+".Get", 0).Store(encodedData)
	if err != nil {
		err = t.processError(err)
	}
	return err
}

func (t *dbusTransport) StoreStateByModelInto(
	modelName string, encodedData *string,
) error {
	obj := t.conn.Object(modelName, "/state")
	err := obj.Call(readDBusInterface+".Get", 0).Store(encodedData)
	if err != nil {
		err = t.processError(err)
	}
	return err
}

func (t *dbusTransport) Export(object transportObject) error {
	if !object.IsValid() {
		if err, ok := object.(error); ok {
			return err
		}
		return errors.New("invalid object")
	}
	switch object.Type() {
	case "config":
		return t.exportConfigInterfaces(object)
	case "state":
		return t.exportStateInterfaces(object)
	case "rpc":
		return t.exportRPCInterfaces(object)
	}
	return nil
}

func (t *dbusTransport) Close() error {
	t.removeAllSubscribers()
	if t.conn == nil {
		return nil
	}
	err := t.conn.Close()
	t.conn = nil
	t.busMgr = nil
	return err
}

func (t *dbusTransport) mapMethodNames(
	methods map[string]interface{},
	mapfn func(string) string,
) map[string]interface{} {
	out := make(map[string]interface{}, len(methods))
	for name, method := range methods {
		out[mapfn(name)] = method
	}
	return out
}

func (t *dbusTransport) exportConfigInterfaces(object transportObject) error {
	busObj := t.busMgr.NewObjectFromTable(
		dbus.ObjectPath("/"+object.Name()),
		t.mapMethodNames(object.Methods(), t.convertYangNameToDBus))
	err := busObj.Implements(readDBusInterface, (*dbusServiceRead)(nil))
	if err != nil {
		return err
	}
	return busObj.Implements(writeDBusInterface, (*dbusServiceWrite)(nil))
}

func (t *dbusTransport) exportStateInterfaces(object transportObject) error {
	busObj := t.busMgr.NewObjectFromTable(
		dbus.ObjectPath("/"+object.Name()),
		t.mapMethodNames(object.Methods(), t.convertYangNameToDBus))
	return busObj.Implements(readDBusInterface, (*dbusServiceRead)(nil))
}

func (t *dbusTransport) exportRPCInterfaces(object transportObject) error {
	intfName := t.getModuleRPCInterfaceName(object.Name())
	methods := t.mapMethodNames(object.Methods(), t.convertYangNameToDBus)
	busObj := t.busMgr.NewObjectFromTable(
		t.getModuleRPCObjectPath(object.Name()), methods)
	return busObj.ImplementsTable(intfName, methods)
}

func (t *dbusTransport) getModuleRPCInterfaceName(moduleName string) string {
	return yangModuleDBusPfx + "." +
		t.convertYangNameToDBus(moduleName) + ".RPC"
}

func (t *dbusTransport) getModuleNotificationInterfaceName(
	moduleName string,
) string {
	return yangModuleDBusPfx + "." +
		t.convertYangNameToDBus(moduleName) + ".Notification"
}

func (t *dbusTransport) getModuleRPCObjectPath(
	moduleName string,
) dbus.ObjectPath {
	return dbus.ObjectPath("/" +
		strings.Replace(moduleName, "-", "_", -1) + "/rpc")
}
func (t *dbusTransport) getModuleNotificationObjectPath(
	moduleName string,
) dbus.ObjectPath {
	return dbus.ObjectPath("/" +
		strings.Replace(moduleName, "-", "_", -1) + "/notification")
}

func (t *dbusTransport) convertYangNameToDBus(name string) string {
	var afterHyphen bool
	var buf []byte
	b := bytes.NewBuffer(buf)
	for i, r := range name {
		if r == '-' {
			afterHyphen = true
			continue
		} else if i == 0 || afterHyphen {
			b.WriteRune(unicode.ToUpper(r))
			afterHyphen = false
		} else {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

func (t *dbusTransport) getDestinationByModuleName(
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
	out, err = t.callYangdRPC("lookup-rpc-destination-by-module-name", data)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err = marshaller.Unmarshal(out, &result); err != nil {
		return "", mgmterror.NewMalformedMessageError()
	}

	return result[yangdModuleName+":destination"].(string), nil
}

func (t *dbusTransport) genYangdRPCMethodName(rpcName string) string {
	return t.getModuleRPCInterfaceName(yangdModuleName) +
		"." + t.convertYangNameToDBus(rpcName)
}

func (t *dbusTransport) processErrorIgnoreUnsupported(err error) error {
	return t.processErrorInternal(err, true)
}

func (t *dbusTransport) processError(err error) error {
	return t.processErrorInternal(err, false)
}

func (t *dbusTransport) processErrorInternal(
	err error,
	ignoreUnsupported bool,
) error {
	if dbuserr, ok := err.(dbus.Error); ok {
		// Convert NoSuchObject errors to a non-DBUS-specific type. Callers
		// can specify if they wish to completely ignore this error, eg for an
		// optional method on the bus.
		if dbuserr.Name == "org.freedesktop.DBus.Error.NoSuchObject" {
			if ignoreUnsupported {
				return nil
			}
			return mgmterror.NewOperationNotSupportedApplicationError()
		}
		rpcerr := getRpcError(dbuserr.Name, dbuserr.Body)
		if rpcerr != nil {
			return rpcerr
		}
	}
	return err
}
func (t *dbusTransport) callYangdRPC(name, input string) (string, error) {
	var output string
	obj := t.conn.Object(yangdName, yangdRPCPath)
	err := obj.Call(t.genYangdRPCMethodName(name), 0, input).Store(&output)
	if err != nil {
		err = t.processError(err)
	}
	return output, err
}

func (t *dbusTransport) isDBusRPC(
	modelName, moduleName, dbusRPCName string,
) bool {
	obj := t.conn.Object(modelName, t.getModuleRPCObjectPath(moduleName))
	node, err := introspect.Call(obj)
	if err != nil {
		return false
	}
	for _, iface := range node.Interfaces {
		if iface.Name == t.getModuleRPCInterfaceName(moduleName) {
			return t.findMethod(iface.Methods, dbusRPCName)
		}
	}
	return false
}

func (t *dbusTransport) findMethod(
	methods []introspect.Method,
	name string,
) bool {
	for _, method := range methods {
		if method.Name == name {
			return true
		}
	}
	return false
}

func (t *dbusTransport) addSubscriber(
	name string,
	subscriber transportSubscriber,
) {
	t.signalHandlers.mu.Lock()
	defer t.signalHandlers.mu.Unlock()

	subs := t.signalHandlers.handlers[name]
	for _, sub := range subs {
		if sub == subscriber {
			return
		}
	}

	subs = append(subs, subscriber)
	t.signalHandlers.handlers[name] = subs
}

func (t *dbusTransport) removeSubscriber(
	name string,
	subscriber transportSubscriber,
) int {
	t.signalHandlers.mu.Lock()
	defer t.signalHandlers.mu.Unlock()
	subs := t.signalHandlers.handlers[name]
	newSubs := make([]transportSubscriber, 0, len(subs))
	for _, sub := range subs {
		if sub == subscriber {
			continue
		}
		newSubs = append(newSubs, sub)
	}
	t.signalHandlers.handlers[name] = newSubs
	return len(newSubs)
}

func (t *dbusTransport) removeAllSubscribers() {
	t.signalHandlers.mu.Lock()
	defer t.signalHandlers.mu.Unlock()
	t.signalHandlers.handlers = make(map[string][]transportSubscriber)
}

// Extracts an embedded MgmtError from a dbus.Error body
func getRpcError(name string, body []interface{}) *mgmterror.MgmtError {
	const errpfx = "com.vyatta.rpcerror."
	var re mgmterror.MgmtError
	if !strings.HasPrefix(name, errpfx) {
		return nil // not a MgmtError
	}
	if err := dbus.Store(body, &re); err != nil {
		return nil // malformed MgmtError
	}
	return &re
}
