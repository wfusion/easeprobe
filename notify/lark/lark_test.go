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

package lark

import (
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wfusion/easeprobe/global"
	"github.com/wfusion/easeprobe/report"
	"github.com/wfusion/gofusion/common/utils/gomonkey"
)

func assertError(t *testing.T, err error, msg string) {
	assert.Error(t, err)
	assert.Equal(t, msg, err.Error())
}
func TestLark(t *testing.T) {
	conf := &NotifyConfig{}
	err := conf.Config(global.NotifySettings{})
	assert.NoError(t, err)
	assert.Equal(t, report.Lark, conf.NotifyFormat)
	assert.Equal(t, "lark", conf.Kind())

	var client *http.Client
	defer gomonkey.ApplyMethod(reflect.TypeOf(client), "Do", func(_ *http.Client, req *http.Request) (*http.Response, error) {
		r := io.NopCloser(strings.NewReader(`{"StatusCode": 0}`))
		return &http.Response{
			StatusCode: 200,
			Body:       r,
		}, nil
	}).Reset()
	err = conf.SendLark("title", "message")
	assert.NoError(t, err)

	// bad response
	defer gomonkey.ApplyMethod(reflect.TypeOf(client), "Do", func(_ *http.Client, req *http.Request) (*http.Response, error) {
		r := io.NopCloser(strings.NewReader(`{"StatusCode": "100"}`))
		return &http.Response{
			StatusCode: 200,
			Body:       r,
		}, nil
	}).Reset()
	err = conf.SendLark("title", "message")
	assertError(t, err, "Error response from Lark - code [0] - msg []")

	defer gomonkey.ApplyMethod(reflect.TypeOf(client), "Do", func(_ *http.Client, req *http.Request) (*http.Response, error) {
		r := io.NopCloser(strings.NewReader(`{"StatusCode": "100", "code": 10, "msg": "lark error"}`))
		return &http.Response{
			StatusCode: 200,
			Body:       r,
		}, nil
	}).Reset()
	err = conf.SendLark("title", "message")
	assertError(t, err, "Error response from Lark - code [10] - msg [lark error]")

	defer gomonkey.ApplyMethod(reflect.TypeOf(client), "Do", func(_ *http.Client, req *http.Request) (*http.Response, error) {
		r := io.NopCloser(strings.NewReader(`bad : json format`))
		return &http.Response{
			StatusCode: 200,
			Body:       r,
		}, nil
	}).Reset()
	err = conf.SendLark("title", "message")
	assertError(t, err, "Error response from Lark [200] - [bad : json format]")

	defer gomonkey.ApplyFunc(io.ReadAll, func(_ io.Reader) ([]byte, error) {
		return nil, errors.New("read error")
	}).Reset()
	err = conf.SendLark("title", "message")
	assertError(t, err, "read error")

	defer gomonkey.ApplyMethod(reflect.TypeOf(client), "Do", func(_ *http.Client, req *http.Request) (*http.Response, error) {
		return nil, errors.New("http error")
	}).Reset()
	err = conf.SendLark("title", "message")
	assertError(t, err, "http error")

	defer gomonkey.ApplyFunc(http.NewRequest, func(_ string, _ string, _ io.Reader) (*http.Request, error) {
		return nil, errors.New("new request error")
	}).Reset()
	err = conf.SendLark("title", "message")
	assertError(t, err, "new request error")

}
