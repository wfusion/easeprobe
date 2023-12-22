/*
 * Copyright (c) 2022, MegaEase
 * All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package zookeeper

import (
	"crypto/tls"
	"fmt"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wfusion/easeprobe/global"
	"github.com/wfusion/easeprobe/probe/client/conf"
	"github.com/wfusion/gofusion/common/utils/gomonkey"

	"github.com/go-zookeeper/zk"
)

func TestZooKeeper(t *testing.T) {
	conf := conf.Options{
		Host:       "127.0.0.0:2181",
		DriverType: conf.Zookeeper,
		Username:   "username",
		Password:   "password",
		TLS: global.TLS{
			CA:   "ca",
			Cert: "cert",
			Key:  "key",
		},
	}

	z, e := New(conf)
	assert.Nil(t, z)
	assert.NotNil(t, e)
	assert.Contains(t, e.Error(), "TLS Config Error")

	conf.TLS = global.TLS{}
	z, e = New(conf)
	assert.NotNil(t, z)
	assert.Nil(t, e)
	assert.Equal(t, "ZooKeeper", z.Kind())

	defer gomonkey.ApplyFunc(net.DialTimeout, func(network, address string, timeout time.Duration) (net.Conn, error) {
		return &net.TCPConn{}, nil
	}).Reset()

	var conn *zk.Conn
	defer gomonkey.ApplyMethod(reflect.TypeOf(conn), "Get", func(_ *zk.Conn, path string) ([]byte, *zk.Stat, error) {
		return []byte("test"), &zk.Stat{}, nil
	}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(conn), "Close", func(_ *zk.Conn) {
		return
	}).Reset()

	defer gomonkey.ApplyFunc(zk.ConnectWithDialer, func(servers []string, sessionTimeout time.Duration, dialer zk.Dialer) (*zk.Conn, <-chan zk.Event, error) {
		return &zk.Conn{}, nil, nil
	}).Reset()
	s, m := z.Probe()
	assert.True(t, s)
	assert.Contains(t, m, "Successfully")

	// TLS config success
	var tc *global.TLS
	defer gomonkey.ApplyMethod(reflect.TypeOf(tc), "Config", func(_ *global.TLS) (*tls.Config, error) {
		return &tls.Config{}, nil
	}).Reset()
	z, e = New(conf)
	assert.NotNil(t, z)
	assert.Nil(t, e)
	assert.NotNil(t, z.tls)

	s, m = z.Probe()
	assert.True(t, s)
	assert.Contains(t, m, "Successfully")

	// Get error
	defer gomonkey.ApplyMethod(reflect.TypeOf(conn), "Get", func(_ *zk.Conn, path string) ([]byte, *zk.Stat, error) {
		return nil, nil, fmt.Errorf("get error")
	}).Reset()
	s, m = z.Probe()
	assert.False(t, s)
	assert.Contains(t, m, "get error")

	// Connect error
	defer gomonkey.ApplyFunc(zk.ConnectWithDialer, func(servers []string, sessionTimeout time.Duration, dialer zk.Dialer) (*zk.Conn, <-chan zk.Event, error) {
		return nil, nil, fmt.Errorf("connect error")
	}).Reset()
	s, m = z.Probe()
	assert.False(t, s)
	assert.Contains(t, m, "connect error")

}

func TestGetDialer(t *testing.T) {
	zConf := &Zookeeper{
		Options: conf.Options{
			Host:       "127.0.0.0:2181",
			DriverType: conf.Redis,
			Username:   "username",
			Password:   "password",
			TLS: global.TLS{
				CA:   "ca",
				Cert: "cert",
				Key:  "key",
			},
		},
		tls: &tls.Config{},
	}

	fn := getDialer(zConf)

	defer gomonkey.ApplyFunc(net.DialTimeout, func(network, address string, timeout time.Duration) (net.Conn, error) {
		return &net.TCPConn{}, nil
	}).Reset()
	var tlsConn *tls.Conn
	defer gomonkey.ApplyMethod(reflect.TypeOf(tlsConn), "Handshake", func(_ *tls.Conn) error {
		return nil
	}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(tlsConn), "Close", func(_ *tls.Conn) error {
		return nil
	}).Reset()

	conn, err := fn("tcp", zConf.Host, time.Second)
	assert.Nil(t, err)
	assert.NotNil(t, conn)

	defer gomonkey.ApplyMethod(reflect.TypeOf(tlsConn), "Handshake", func(_ *tls.Conn) error {
		return fmt.Errorf("handshake error")
	}).Reset()
	conn, err = fn("tcp", zConf.Host, time.Second)
	assert.Equal(t, "handshake error", err.Error())
	assert.Nil(t, conn)

	defer gomonkey.ApplyFunc(net.DialTimeout, func(network, address string, timeout time.Duration) (net.Conn, error) {
		return nil, fmt.Errorf("dial error")
	}).Reset()
	conn, err = fn("tcp", zConf.Host, time.Second)
	assert.Equal(t, "dial error", err.Error())
	assert.Nil(t, conn)
}

func TestData(t *testing.T) {
	z := &Zookeeper{
		Options: conf.Options{
			Host:       "127.0.0.0:2181",
			DriverType: conf.Redis,
			Username:   "username",
			Password:   "password",
			Data: map[string]string{
				"test": "test",
			},
		},
	}

	defer gomonkey.ApplyFunc(getDialer, func(z *Zookeeper) func(string, string, time.Duration) (net.Conn, error) {
		return net.DialTimeout
	}).Reset()
	defer gomonkey.ApplyFunc(zk.ConnectWithDialer, func(servers []string, sessionTimeout time.Duration, dialer zk.Dialer) (*zk.Conn, <-chan zk.Event, error) {
		return &zk.Conn{}, nil, nil
	}).Reset()
	var conn *zk.Conn
	defer gomonkey.ApplyMethod(reflect.TypeOf(conn), "Close", func(_ *zk.Conn) {
		return
	}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(conn), "Get", func(_ *zk.Conn, path string) ([]byte, *zk.Stat, error) {
		return []byte("test"), &zk.Stat{}, nil
	}).Reset()

	s, m := z.Probe()
	assert.True(t, s)
	assert.Contains(t, m, "Successfully")

	z.Data = map[string]string{
		"test": "test1",
	}
	s, m = z.Probe()
	assert.False(t, s)
	assert.Contains(t, m, "Data not match")

	defer gomonkey.ApplyMethod(reflect.TypeOf(conn), "Get", func(_ *zk.Conn, path string) ([]byte, *zk.Stat, error) {
		return []byte(""), &zk.Stat{}, fmt.Errorf("get error")
	}).Reset()
	s, m = z.Probe()
	assert.False(t, s)
	assert.Contains(t, m, "get error")

}
