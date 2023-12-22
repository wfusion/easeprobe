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

package redis

import (
	"context"
	"crypto/tls"
	"fmt"
	"reflect"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/wfusion/easeprobe/global"
	"github.com/wfusion/easeprobe/probe/client/conf"
	"github.com/wfusion/gofusion/common/utils/gomonkey"
)

func TestRedis(t *testing.T) {
	conf := conf.Options{
		Host:       "example.com",
		DriverType: conf.Redis,
		Username:   "username",
		Password:   "password",
		TLS: global.TLS{
			CA:   "ca",
			Cert: "cert",
			Key:  "key",
		},
	}

	r, err := New(conf)
	assert.Nil(t, r)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "TLS Config Error")

	conf.TLS = global.TLS{}
	r, err = New(conf)
	assert.NotNil(t, r)
	assert.Nil(t, err)
	assert.Equal(t, "Redis", r.Kind())
	assert.Nil(t, r.tls)

	var client *redis.Client
	defer gomonkey.ApplyMethod(reflect.TypeOf(client), "Ping", func(_ *redis.Client, ctx context.Context) *redis.StatusCmd {
		return &redis.StatusCmd{}
	}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(client), "Close", func(_ *redis.Client) error {
		return nil
	}).Reset()
	var statusCmd *redis.StatusCmd
	defer gomonkey.ApplyMethod(reflect.TypeOf(statusCmd), "Result", func(_ *redis.StatusCmd) (string, error) {
		return "PONG", nil
	}).Reset()

	s, m := r.Probe()
	assert.True(t, s)
	assert.Contains(t, m, "Successfully")

	// TLS config success
	var tc *global.TLS
	defer gomonkey.ApplyMethod(reflect.TypeOf(tc), "Config", func(_ *global.TLS) (*tls.Config, error) {
		return &tls.Config{}, nil
	}).Reset()

	r, err = New(conf)
	assert.NotNil(t, r)
	assert.Nil(t, err)
	assert.NotNil(t, r.tls)

	s, m = r.Probe()
	assert.True(t, s)
	assert.Contains(t, m, "Successfully")

	// Ping error
	defer gomonkey.ApplyMethod(reflect.TypeOf(statusCmd), "Result", func(_ *redis.StatusCmd) (string, error) {
		return "", fmt.Errorf("ping error")
	}).Reset()
	s, m = r.Probe()
	assert.False(t, s)
	assert.Contains(t, m, "ping error")

}

func TestData(t *testing.T) {
	conf := conf.Options{
		Host:       "example.com",
		DriverType: conf.Redis,
		Username:   "username",
		Password:   "password",
		Data: map[string]string{
			"key1": "value1",
		},
	}

	r, err := New(conf)
	assert.NotNil(t, r)
	assert.Nil(t, err)

	var client *redis.Client
	defer gomonkey.ApplyMethod(reflect.TypeOf(client), "Get", func(_ *redis.Client, ctx context.Context, key string) *redis.StringCmd {
		return &redis.StringCmd{}
	}).Reset()
	var cmd *redis.StringCmd
	defer gomonkey.ApplyMethod(reflect.TypeOf(cmd), "Result", func(_ *redis.StringCmd) (string, error) {
		return "value1", nil
	}).Reset()

	s, m := r.Probe()
	assert.True(t, s)
	assert.Contains(t, m, "Successfully")

	defer gomonkey.ApplyMethod(reflect.TypeOf(cmd), "Result", func(_ *redis.StringCmd) (string, error) {
		return "value", nil
	}).Reset()
	s, m = r.Probe()
	assert.False(t, s)
	assert.Contains(t, m, "value")

	defer gomonkey.ApplyMethod(reflect.TypeOf(cmd), "Result", func(_ *redis.StringCmd) (string, error) {
		return "", fmt.Errorf("get result error")
	}).Reset()
	s, m = r.Probe()
	assert.False(t, s)
	assert.Contains(t, m, "get result error")

}
