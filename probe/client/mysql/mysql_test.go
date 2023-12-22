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

package mysql

import (
	"crypto/tls"
	"database/sql"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/wfusion/easeprobe/global"
	"github.com/wfusion/easeprobe/probe/client/conf"
	"github.com/wfusion/gofusion/common/utils/gomonkey"
)

func TestMySQL(t *testing.T) {
	conf := conf.Options{
		Host:       "example.com",
		DriverType: conf.MySQL,
		Username:   "username",
		Password:   "password",
		TLS: global.TLS{
			CA:   "ca",
			Cert: "cert",
			Key:  "key",
		},
	}

	my, err := New(conf)
	assert.Nil(t, my)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "TLS Config Error")

	conf.TLS = global.TLS{}
	my, err = New(conf)
	assert.Nil(t, err)
	assert.Equal(t, "MySQL", my.Kind())
	connStr := fmt.Sprintf("%s:%s@tcp(%s)/?timeout=%s",
		conf.Username, conf.Password, conf.Host, conf.Timeout().Round(time.Second))
	assert.Equal(t, connStr, my.ConnStr)

	conf.Password = ""
	my, err = New(conf)
	connStr = fmt.Sprintf("%s@tcp(%s)/?timeout=%s",
		conf.Username, conf.Host, conf.Timeout().Round(time.Second))
	assert.Equal(t, connStr, my.ConnStr)

	defer gomonkey.ApplyFunc(sql.Open, func(driverName, dataSourceName string) (*sql.DB, error) {
		return &sql.DB{}, nil
	}).Reset()
	var db *sql.DB
	defer gomonkey.ApplyMethod(reflect.TypeOf(db), "Close", func(_ *sql.DB) error {
		return nil
	}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(db), "Ping", func(_ *sql.DB) error {
		return nil
	}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(db), "Query", func(_ *sql.DB, query string, args ...interface{}) (*sql.Rows, error) {
		return &sql.Rows{}, nil
	}).Reset()
	var r *sql.Rows
	defer gomonkey.ApplyMethod(reflect.TypeOf(r), "Close", func(_ *sql.Rows) error {
		return nil
	}).Reset()

	s, m := my.Probe()
	assert.True(t, s)
	assert.Contains(t, m, "Successfully")

	// TLS config success
	var tc *global.TLS
	defer gomonkey.ApplyMethod(reflect.TypeOf(tc), "Config", func(_ *global.TLS) (*tls.Config, error) {
		return &tls.Config{}, nil
	}).Reset()
	defer gomonkey.ApplyFunc(mysql.RegisterTLSConfig, func(name string, config *tls.Config) error {
		return nil
	}).Reset()

	my, err = New(conf)
	assert.NotNil(t, my.tls)
	s, m = my.Probe()
	assert.True(t, s)
	assert.Contains(t, m, "Successfully")

	//  Query error
	defer gomonkey.ApplyMethod(reflect.TypeOf(db), "Query", func(_ *sql.DB, query string, args ...interface{}) (*sql.Rows, error) {
		return nil, fmt.Errorf("query error")
	}).Reset()
	s, m = my.Probe()
	assert.False(t, s)
	assert.Contains(t, m, "query error")

	// Ping error
	defer gomonkey.ApplyMethod(reflect.TypeOf(db), "Ping", func(_ *sql.DB) error {
		return fmt.Errorf("ping error")
	}).Reset()
	s, m = my.Probe()
	assert.False(t, s)
	assert.Contains(t, m, "ping error")

	// Connect error
	defer gomonkey.ApplyFunc(sql.Open, func(driverName, dataSourceName string) (*sql.DB, error) {
		return nil, fmt.Errorf("connect error")
	}).Reset()
	s, m = my.Probe()
	assert.False(t, s)
	assert.Contains(t, m, "connect error")

}

func TestData(t *testing.T) {
	defer gomonkey.ApplyFunc(sql.Open, func(driverName, dataSourceName string) (*sql.DB, error) {
		return &sql.DB{}, nil
	}).Reset()
	var db *sql.DB
	defer gomonkey.ApplyMethod(reflect.TypeOf(db), "Close", func(_ *sql.DB) error {
		return nil
	}).Reset()

	conf := conf.Options{
		Host:       "example.com",
		DriverType: conf.MySQL,
		Username:   "username",
		Password:   "password",
		Data: map[string]string{
			"": "",
		},
	}
	my, err := New(conf)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Empty SQL data")

	conf.Data = map[string]string{
		"key": "value",
	}
	my, err = New(conf)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Invalid SQL data")

	conf.Data = map[string]string{
		"database:table:column:key:value": "expected",
	}
	my, err = New(conf)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "the value must be int")

	defer gomonkey.ApplyMethod(reflect.TypeOf(db), "Ping", func(_ *sql.DB) error {
		return nil
	}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(db), "Query", func(_ *sql.DB, query string, args ...interface{}) (*sql.Rows, error) {
		return &sql.Rows{}, nil
	}).Reset()
	var r *sql.Rows
	defer gomonkey.ApplyMethod(reflect.TypeOf(r), "Close", func(_ *sql.Rows) error {
		return nil
	}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(r), "Next", func(_ *sql.Rows) bool {
		return true
	}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(r), "Scan", func(_ *sql.Rows, args ...interface{}) error {
		v := args[0].(*string)
		*v = "expected"
		return nil
	}).Reset()

	conf.Data = map[string]string{
		"database:table:column:key:1": "expected",
	}
	my, err = New(conf)
	s, m := my.Probe()
	assert.True(t, s)
	assert.Contains(t, m, "Successfully")

	//mismatch
	defer gomonkey.ApplyMethod(reflect.TypeOf(r), "Scan", func(_ *sql.Rows, args ...interface{}) error {
		v := args[0].(*string)
		*v = "unexpected"
		return nil
	}).Reset()
	s, m = my.Probe()
	assert.False(t, s)
	assert.Contains(t, m, "alue not match")

	// scan error
	defer gomonkey.ApplyMethod(reflect.TypeOf(r), "Scan", func(_ *sql.Rows, args ...interface{}) error {
		return fmt.Errorf("scan error")
	}).Reset()
	s, m = my.Probe()
	assert.False(t, s)
	assert.Contains(t, m, "scan error")

	// Next error
	defer gomonkey.ApplyMethod(reflect.TypeOf(r), "Next", func(_ *sql.Rows) bool {
		return false
	}).Reset()
	s, m = my.Probe()
	assert.False(t, s)
	assert.Contains(t, m, "No data")

	// Query error
	defer gomonkey.ApplyMethod(reflect.TypeOf(db), "Query", func(_ *sql.DB, query string, args ...interface{}) (*sql.Rows, error) {
		return nil, fmt.Errorf("query error")
	}).Reset()
	s, m = my.Probe()
	assert.False(t, s)
	assert.Contains(t, m, "query error")

	my.Data = map[string]string{
		"key": "value",
	}
	s, m = my.Probe()
	assert.False(t, s)
	assert.Contains(t, m, "Invalid SQL data")

}
