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

package sms

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wfusion/easeprobe/global"
	"github.com/wfusion/easeprobe/notify/sms/conf"
	"github.com/wfusion/easeprobe/notify/sms/nexmo"
	"github.com/wfusion/easeprobe/notify/sms/twilio"
	"github.com/wfusion/easeprobe/notify/sms/yunpian"
	"github.com/wfusion/gofusion/common/utils/gomonkey"
)

func TestSMS(t *testing.T) {
	c := &NotifyConfig{}
	c.NotifyName = "dummy"
	c.ProviderType = conf.Yunpian
	err := c.Config(global.NotifySettings{})
	assert.NoError(t, err)
	assert.IsType(t, &yunpian.Yunpian{}, c.Provider)

	c.ProviderType = conf.Twilio
	c.Provider = nil
	err = c.Config(global.NotifySettings{})
	assert.NoError(t, err)
	assert.IsType(t, &twilio.Twilio{}, c.Provider)

	c.ProviderType = conf.Nexmo
	c.Provider = nil
	err = c.Config(global.NotifySettings{})
	assert.NoError(t, err)
	assert.IsType(t, &nexmo.Nexmo{}, c.Provider)

	var provider *nexmo.Nexmo
	defer gomonkey.ApplyMethod(reflect.TypeOf(provider), "Notify", func(_ *nexmo.Nexmo, _ string, _ string) error {
		return nil
	}).Reset()
	err = c.DoNotify("title", "message")
	assert.NoError(t, err)

	c.ProviderType = conf.Unknown
	c.Provider = nil
	err = c.Config(global.NotifySettings{})
	assert.NoError(t, err)
	assert.Nil(t, c.Provider)

	err = c.DoNotify("title", "text")
	assert.Error(t, err)
	assert.Equal(t, "wrong Provider type", err.Error())

}
