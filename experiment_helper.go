package sensorsabtest

import (
	"fmt"
	"github.com/sensorsdata/abtesting-sdk-go/beans"
	"github.com/sensorsdata/abtesting-sdk-go/utils"
	"github.com/sensorsdata/abtesting-sdk-go/utils/lru"
	"reflect"
	"sync"
	"time"
)

// 用户的试验缓存
var experimentCache = lru.New(4096)
var experimentLock = sync.Mutex{}

// $ABTestTrigger 事件缓存
var eventsCache = lru.New(4096)
var eventsLock = sync.Mutex{}

// 插件版本号标记位
var isFirstEvent = true

func loadExperimentFromNetwork(sensors *SensorsABTest, requestParam beans.RequestParam, defaultValue interface{}, isTrack bool) (error error, variable interface{}, experiment beans.Experiment) {
	experiments, err := utils.RequestExperiment(sensors.config.APIUrl, buildRequestParam(requestParam), time.Duration(sensors.config.Timeout)*time.Millisecond)
	if err != nil {
		return err, defaultValue, beans.Experiment{}
	}

	distinctId, isLoginId := getDistinctId(requestParam.LoginId, requestParam.AnonymousId)
	var experimentParam = requestParam.ExperimentParam
	// 遍历试验
	for _, experiment := range experiments {
		// 遍历试验变量
		for _, variable := range experiment.VariableList {
			if experimentParam == variable.Name && isEqualType(defaultValue, variable) {
				if isTrack {
					trackABTestEvent(distinctId, isLoginId, experiment, sensors, nil)
					saveEvent2Cache(distinctId, experiment, sensors)
				}
				// 回调试验变量给客户
				return nil, variable.Value, experiment
			}
		}
	}

	return nil, nil, beans.Experiment{}
}

func loadExperimentFromCache(sensors *SensorsABTest, requestParam beans.RequestParam, defaultValue interface{}, isTrack bool) (error error, variable interface{}, experiment beans.Experiment) {
	var tempVariable interface{}
	distinctId, isLoginId := getDistinctId(requestParam.LoginId, requestParam.AnonymousId)
	tempExperiment, ok := loadExperimentCache(distinctId)
	if tempExperiment == nil && ok {
		error, tempVariable, tempExperiment = loadExperimentFromNetwork(sensors, requestParam, defaultValue, isTrack)
		if error != nil {
			return error, defaultValue, beans.Experiment{}
		}
		// 缓存试验
		saveExperiment2Cache(distinctId, tempExperiment.(beans.Experiment), sensors.config.ExperimentCacheTime)
	}

	te, ok := tempExperiment.(beans.Experiment)
	if ok {
		trackABTestEvent(distinctId, isLoginId, te, sensors, nil)
	}
	return nil, tempVariable, te
}

func trackABTestEvent(distinctId string, isLoginId bool, experiment beans.Experiment, sensors *SensorsABTest, properties map[string]interface{}) {
	// 是对照组，则不触发 $ABTestTrigger 事件
	if experiment.IsControlGroup {
		return
	}

	// 如果在缓存中，则不触发 $ABTestTrigger 事件
	_, ok := loadEventFromCache(distinctId)
	if ok {
		return
	}

	if properties == nil {
		properties = map[string]interface{}{
			"abtest_experiment_id":       experiment.AbtestExperimentId,
			"abtest_experiment_group_id": experiment.AbtestExperimentGroupId,
		}
	} else {
		properties["abtest_experiment_id"] = experiment.AbtestExperimentId
		properties["abtest_experiment_group_id"] = experiment.AbtestExperimentGroupId
	}

	if isFirstEvent {
		properties["$lib_plugin_version"] = []string{"golang_abtesting:" + SDK_VERSION}
		isFirstEvent = false
	}

	err := sensors.sensorsAnalytics.Track(distinctId, "$ABTestTrigger", properties, isLoginId)
	if err != nil {
		fmt.Println("$ABTestTrigger track failed.")
	}
}

// 从缓存读取 $ABTestTrigger
func loadEventFromCache(distinctId string) (interface{}, bool) {
	eventsLock.Lock()
	defer eventsLock.Unlock()
	return eventsCache.Get(distinctId)
}

// 保存 $ABTestTrigger 到缓存中
func saveEvent2Cache(distinctId string, experiment beans.Experiment, sensors *SensorsABTest) {
	// 缓存 $ABTestTrigger 事件
	if sensors.config.EnableEventCache {
		eventsLock.Lock()
		defer eventsLock.Unlock()
		eventsCache.Add(distinctId, experiment)
		// 进行清理缓存
		removeCache(distinctId, func(id string) {
			eventsCache.Remove(id)
		}, sensors.config.EventCacheTime)
	}
}

// 从缓存读取试验
func loadExperimentCache(distinctId string) (interface{}, bool) {
	experimentLock.Lock()
	defer experimentLock.Unlock()
	return experimentCache.Get(distinctId)
}

// 保存试验到缓存
func saveExperiment2Cache(distinctId string, experiment beans.Experiment, timeout time.Duration) {
	experimentLock.Lock()
	defer experimentLock.Unlock()
	experimentCache.Add(distinctId, experiment)
	// 进行清理缓存
	removeCache(distinctId, func(id string) {
		eventsCache.Remove(id)
	}, timeout)
}

func getDistinctId(loginId string, anonymousId string) (string, bool) {
	var distinctId string
	var isLoginId bool
	if distinctId == "" {
		distinctId = loginId
		isLoginId = true
	}

	if distinctId == "" {
		distinctId = anonymousId
		isLoginId = false
	}
	return distinctId, isLoginId
}

func isEqualType(defaultValue interface{}, variables beans.Variables) bool {
	var defaultType = reflect.TypeOf(defaultValue)
	if variables.Type == "STRING" && "string" == defaultType.String() {
		return true
	} else if variables.Type == "INTEGER" && ("int" == defaultType.String() || "int8" == defaultType.String() ||
		"int16" == defaultType.String() || "int32" == defaultType.String() || "int64" == defaultType.String()) {
		return true
	} else if variables.Type == "BOOLEAN" && "bool" == defaultType.String() {
		return true
	}
	return false
}

// 拼接网络请求参数
func buildRequestParam(requestParam beans.RequestParam) map[string]interface{} {
	var params = make(map[string]interface{})
	if requestParam.LoginId != "" {
		params["login_id"] = requestParam.LoginId
	}

	if requestParam.AnonymousId != "" {
		params["anonymous_id"] = requestParam.LoginId
	}

	params["abtest_lib_version"] = SDK_VERSION
	params["platform"] = LIB_NAME
	params["properties"] = requestParam.HttpRequestPrams
	return params
}

// 清理缓存
func removeCache(distinctId string, removeCache func(id string), timeout time.Duration) {
	go func() {
		d := timeout * time.Second
		t := time.NewTicker(d)
		defer t.Stop()
		<-t.C
		removeCache(distinctId)
	}()
}
