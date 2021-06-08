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
	"time"
)

func main() {
	// 初始化埋点 SDK，使用 BatchConsumer
	//consumer, _ := sensorsanalytics.InitBatchConsumer("http://10.130.6.4:8106/sa?project=default", 1, 5)
	// 初始化埋点 SDK，使用 ConcurrentLoggingConsumer
	consumer, _ := sensorsanalytics.InitConcurrentLoggingConsumer("./log.data", false)
	sa := sensorsanalytics.InitSensorsAnalytics(consumer, "default", false)

	defer sa.Close()
	// 进行初始化配置
	abconfig := beans.ABTestConfig{
		APIUrl:           "http://abtesting.saas.debugbox.sensorsdata.cn/api/v2/abtest/online/results?project-key=438B9364C98D54371751BA82F6484A1A03A5155E",
		EnableEventCache: false,
		Timeout:          time.Duration(1),
		SensorsAnalytics: sa,
	}
	// 初始化 A/B Testing SDK
	sensorsAB := sensorsabtest.InitSensorsABTest(abconfig)
	requestPara := beans.RequestParam{
		//ExperimentParam:"wz_test_string",
		ParamName: "btn_type",
	}
	// 直接从网络获取试验，由 SDK 触发埋点事件
	_, value := sensorsAB.AsyncFetchABTest("abcd123", true, requestPara, "default")
	fmt.Println("根据试验变量 value 值做试验, value = ", value)

	sa.Flush()
	// 直接从网络获取试验，由调用方触发埋点事件
	_, value, experiment := sensorsAB.AsyncFetchABTestExperiment("abcd123", true, requestPara, "default")
	fmt.Println("根据试验变量 value 值做试验, 并且自己触发埋点。value = ", value)

	// 优先从缓存获取试验，由 SDK 触发埋点事件
	err, value := sensorsAB.FastFetchABTest("abcd123", true, requestPara, 342)
	if err != nil {
		// TODO 根据返回值做试验
	} else {
		// TODO  异常场景下的处理
	}
	fmt.Println("根据试验变量 value 值做试验, value = ", value)

	// 优先从缓存获取试验，并自己触发埋点
	err, value, experiment = sensorsAB.FastFetchABTestExperiment("abcd123", true, requestPara, "test")
	if err == nil {
		// 触发埋点事件
		_ = sensorsAB.TrackABTestTrigger("login123", true, experiment, map[string]interface{}{
			"test":    "test",
			"andoter": "andoter",
			"antway":  "dddddd",
		})
		fmt.Println("根据试验变量 value 值做试验, 并且自己触发埋点。 value = ", value)
	}
}
