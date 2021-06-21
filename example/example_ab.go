/*
 * Created by dengshiwei on 2020/01/06.
 * Copyright 2015－2020 Sensors Data Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"fmt"
	"github.com/sensorsdata/abtesting-sdk-go"
	"github.com/sensorsdata/abtesting-sdk-go/beans"
	"github.com/sensorsdata/sa-sdk-go"
)

func main() {
	// 初始化埋点 SDK，使用 BatchConsumer
	//consumer, _ := sensorsanalytics.InitBatchConsumer("", 1, 5)
	// 初始化埋点 SDK，使用 ConcurrentLoggingConsumer
	consumer, _ := sensorsanalytics.InitConcurrentLoggingConsumer("./log.data", false)
	sa := sensorsanalytics.InitSensorsAnalytics(consumer, "default", false)

	defer sa.Close()
	// 进行初始化配置
	abconfig := beans.ABTestConfig{
		APIUrl:           "",
		EnableEventCache: true,
		SensorsAnalytics: sa,
	}
	// 初始化 A/B Testing SDK
	err, sensorsAB := sensorsabtest.InitSensorsABTest(abconfig)
	if err != nil {
		fmt.Println(err)
	}
	requestPara := beans.RequestParam{
		ParamName:              "o",
		DefaultValue:           "{\"a\":\"Hello\",\"b\":\"World\"}",
		EnableAutoTrackABEvent: true, // 由 SDK 自动触发 A/B Testing 的埋点事件，这样就无需调用端触发了
	}

	// 直接从网络获取试验
	err, experiment := sensorsAB.AsyncFetchABTest("abcd123", true, requestPara)
	fmt.Println("根据试验变量 value 值做试验, value = ", experiment.Result)

	requestPara = beans.RequestParam{
		ParamName:              "btn_type",
		DefaultValue:           "default",
		EnableAutoTrackABEvent: false, // 无需调用端触发 A/B Testing 埋点事件
	}
	// 优先从缓存获取试验，并自己触发埋点
	err, experiment = sensorsAB.FastFetchABTest("abcd123", true, requestPara)
	if err == nil {
		// 触发埋点事件
		_ = sensorsAB.TrackABTestTrigger(experiment, map[string]interface{}{
			"test":    "test",
			"andoter": "andoter",
			"antway":  "dddddd",
		})
		fmt.Println("根据试验变量 value 值做试验, 并且自己触发埋点。 value = ", experiment.Result)
	}
}
