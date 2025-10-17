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

	sensorsabtest "github.com/sensorsdata/abtesting-sdk-go"
	"github.com/sensorsdata/abtesting-sdk-go/beans"
	sensorsanalytics "github.com/sensorsdata/sa-sdk-go"
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

	// FetchAllExperiments 接口使用示例
	demoFetchAllExperiments(sensorsAB)

	// 序列化和反序列化示例
	demoSerializeForTransfer(sensorsAB)
}

// FetchAllExperiments 接口使用示例
func demoFetchAllExperiments(sensorsAB sensorsabtest.SensorsABTest) {
	fmt.Println("\n=== FetchAllExperiments 接口使用示例 ===")

	// 获取用户所有试验结果的请求参数
	requestParam := beans.FetchAllRequestParam{
		Properties: map[string]interface{}{
			"device_type": "mobile",
			"platform":    "android",
			"app_version": "1.2.3",
		},
		CustomIDs: map[string]string{
			"device_id": "device123456",
		},
		TimeoutMilliseconds:    5000,
		EnableAutoTrackABEvent: true, // 开启自动埋点
	}

	// 获取用户所有试验结果
	err, result := sensorsAB.FetchAllExperiments("user123", true, requestParam)

	if err != nil {
		fmt.Printf("获取试验失败: %s\n", err.Error())
		return
	}

	// 如果没有错误，说明成功获取数据
	fmt.Printf("成功获取用户试验数据，用户ID: %s\n", result.DistinctId())

	// 获取具体参数值（会自动埋点）
	buttonColor := result.GetValue("button_color", "blue")
	fmt.Printf("按钮颜色: %v\n", buttonColor)

	algo := result.GetValue("recommendation_algo", "default")
	fmt.Printf("推荐算法: %v\n", algo)

	// 获取不存在的参数（返回默认值）
	pageSize := result.GetValue("page_size", 10)
	fmt.Printf("页面大小: %v (默认值)\n", pageSize)

	fmt.Println("注意：只有调用 GetValue() 方法时才会触发埋点事件")
}

// SerializeForTransfer 序列化示例 - 自动包含用户上下文
func demoSerializeForTransfer(sensorsAB sensorsabtest.SensorsABTest) {
	fmt.Println("\n=== 跨服务传递分流结果示例 ===")
	fmt.Println(" 优势：自动序列化 distinct_id 和 is_login_id，无需手动传递用户信息")

	// 第一步：获取用户所有试验结果
	requestParam := beans.FetchAllRequestParam{
		Properties: map[string]interface{}{
			"device_type": "mobile",
			"platform":    "ios",
		},
		CustomIDs: map[string]string{
			"device_id": "device999",
		},
		TimeoutMilliseconds:    5000,
		EnableAutoTrackABEvent: true,
	}

	err, result := sensorsAB.FetchAllExperiments("user789", true, requestParam)
	if err != nil {
		fmt.Printf("获取试验失败: %s\n", err.Error())
		return
	}

	fmt.Printf("成功获取试验数据，用户ID: %s, 是否登录ID: %v\n", result.DistinctId(), result.IsLoginId())

	// 第二步：使用 Serialize 方法序列化（自动包含 distinct_id、is_login_id 和 custom_ids）
	serializedData, err := result.Dump()
	if err != nil {
		fmt.Printf("序列化失败: %s\n", err.Error())
		return
	}

	fmt.Printf("序列化成功，数据长度: %d 字符\n", len(serializedData))
	fmt.Printf("序列化数据示例（前150字符）: %.150s...\n", serializedData)

	// 模拟跨服务传递序列化数据
	fmt.Println("\n 模拟将序列化数据传递给其他服务...")

	// 第三步：在另一个服务中使用 LoadAllExperimentsFromSerialized 方法反序列化
	loadParam := beans.LoadDumpedParam{
		CustomIDs:              requestParam.CustomIDs, // 传入 CustomIDs 用于校验
		EnableAutoTrackABEvent: true,
	}
	// 注意：这里需要传入 distinct_id、is_login_id 和 param 用于校验
	err, restoredResult := sensorsAB.LoadAllExperiments("user789", true, loadParam, serializedData)
	if err != nil {
		fmt.Printf("反序列化失败: %s\n", err.Error())
		return
	}

	fmt.Printf("反序列化成功，用户ID: %s, 是否登录ID: %v\n", restoredResult.DistinctId(), restoredResult.IsLoginId())

	// 第四步：验证数据完整性
	fmt.Printf("\n 数据验证:\n")
	fmt.Printf("   原始 DistinctId: %s，重建 DistinctId: %s\n", result.DistinctId(), restoredResult.DistinctId())
	fmt.Printf("   原始 IsLoginId: %v，重建 IsLoginId: %v\n", result.IsLoginId(), restoredResult.IsLoginId())

	// 比较参数值
	originalValue := result.GetValue("button_color", nil)
	restoredValue := restoredResult.GetValue("button_color", nil)

	fmt.Printf("\n 测试参数 '%s':\n", "button_color")
	fmt.Printf("   原始值: %v\n", originalValue)
	fmt.Printf("   重建值: %v\n", restoredValue)

	if fmt.Sprintf("%v", originalValue) == fmt.Sprintf("%v", restoredValue) {
		fmt.Println("数据一致性验证通过")
	} else {
		fmt.Println("数据一致性验证失败")
	}

	// 用户上下文验证
	if result.DistinctId() == restoredResult.DistinctId() && result.IsLoginId() == restoredResult.IsLoginId() {
		fmt.Println("用户上下文验证通过")
	} else {
		fmt.Println("用户上下文验证失败")
	}
}
