package sensorsabtest

import (
	"fmt"
	"github.com/sensorsdata/abtesting-sdk-go/beans"
	"github.com/sensorsdata/abtesting-sdk-go/utils"
	"github.com/sensorsdata/abtesting-sdk-go/utils/lru"
	"reflect"
	"strconv"
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

// 埋点事件上次触发的时间
var lastTimeEvent string

func loadExperimentFromNetwork(sensors *SensorsABTest, distinctId string, isLoginId bool, requestParam beans.RequestParam, isTrack bool) (error, interface{}, beans.Experiment) {
	if requestParam.TimeoutMilliseconds <= 0 {
		requestParam.TimeoutMilliseconds = 3 * 1000
	}
	experiments, err := requestExperimentOnNetwork(sensors.config.APIUrl, distinctId, isLoginId, requestParam)
	if err != nil {
		return err, requestParam.DefaultValue, beans.Experiment{}
	}

	variable, experiment := filterExperiment(requestParam, experiments)
	if experiment.AbtestExperimentId != "" {
		if isTrack {
			trackABTestEvent(distinctId, isLoginId, experiment, sensors, nil)
		}
		// 回调试验变量给客户
		experiment.DistinctId = distinctId
		experiment.IsLoginId = isLoginId
		return nil, variable, experiment
	}

	return nil, requestParam.DefaultValue, beans.Experiment{}
}

func loadExperimentFromCache(sensors *SensorsABTest, distinctId string, isLoginId bool, requestParam beans.RequestParam, isTrack bool) (error, interface{}, beans.Experiment) {
	var tempVariable = requestParam.DefaultValue
	var experiment beans.Experiment
	tempExperiments, ok := loadExperimentCache(distinctId)
	if tempExperiments == nil || !ok {
		// 从网络请求试验
		experiments, err := requestExperimentOnNetwork(sensors.config.APIUrl, distinctId, isLoginId, requestParam)
		if err != nil {
			return err, requestParam.DefaultValue, beans.Experiment{}
		}

		// 缓存试验
		saveExperiment2Cache(distinctId, experiments, sensors.config.ExperimentCacheTime)
		// 筛选试验
		variable, experiment := filterExperiment(requestParam, experiments)
		if experiment.AbtestExperimentId != "" {
			if isTrack {
				trackABTestEvent(distinctId, isLoginId, experiment, sensors, nil)
			}
			// 回调试验变量给客户
			experiment.DistinctId = distinctId
			experiment.IsLoginId = isLoginId
			return nil, variable, experiment
		}
	} else {
		tempVariable, experiment = filterExperiment(requestParam, tempExperiments.([]beans.Experiment))
	}

	if isTrack {
		trackABTestEvent(distinctId, isLoginId, experiment, sensors, nil)
	}
	experiment.DistinctId = distinctId
	experiment.IsLoginId = isLoginId
	return nil, tempVariable, experiment
}

// 从网络加载试验
func requestExperimentOnNetwork(apiUrl string, distinctId string, isLoginId bool, requestParam beans.RequestParam) ([]beans.Experiment, error) {
	if requestParam.TimeoutMilliseconds <= 0 {
		requestParam.TimeoutMilliseconds = 3 * 1000
	}
	return utils.RequestExperiment(apiUrl, buildRequestParam(distinctId, isLoginId, requestParam), time.Duration(requestParam.TimeoutMilliseconds)*time.Millisecond)
}

// 筛选试验
func filterExperiment(requestParam beans.RequestParam, experiments []beans.Experiment) (interface{}, beans.Experiment) {
	var experimentParam = requestParam.ParamName
	// 遍历试验
	for _, experiment := range experiments {
		// 遍历试验变量
		for _, variable := range experiment.VariableList {
			ok, value := castValue(requestParam.DefaultValue, variable)
			if experimentParam == variable.Name && ok {
				return value, experiment
			}
		}
	}

	return nil, beans.Experiment{}
}

func trackABTestEvent(distinctId string, isLoginId bool, experiment beans.Experiment, sensors *SensorsABTest, properties map[string]interface{}) {
	if sensors.config.SensorsAnalytics.C == nil {
		return
	}

	// 是白名单，则不触发 $ABTestTrigger 事件
	if experiment.IsWhiteList {
		return
	}

	// 如果在缓存中，则不触发 $ABTestTrigger 事件
	_, ok := loadEventFromCache(distinctId)
	if ok {
		return
	}

	saveEvent2Cache(distinctId, experiment, sensors)
	if properties == nil {
		properties = map[string]interface{}{
			"$abtest_experiment_id":       experiment.AbtestExperimentId,
			"$abtest_experiment_group_id": experiment.AbtestExperimentGroupId,
		}
	} else {
		properties["$abtest_experiment_id"] = experiment.AbtestExperimentId
		properties["$abtest_experiment_group_id"] = experiment.AbtestExperimentGroupId
	}

	currentTime := time.Now().Format("2006-01-02")
	if isFirstEvent || currentTime != lastTimeEvent {
		properties["$lib_plugin_version"] = []string{"golang_abtesting:" + SDK_VERSION}
		isFirstEvent = false
		lastTimeEvent = currentTime
	}

	err := sensors.sensorsAnalytics.Track(distinctId, "$ABTestTrigger", properties, isLoginId)
	if err != nil {
		fmt.Println("$ABTestTrigger track failed, error : ", err)
	}
	sensors.sensorsAnalytics.Flush()
}

// 初始化缓存大小
func initCache(config beans.ABTestConfig) {
	if config.EventCacheSize != 0 {
		eventsCache = lru.New(config.EventCacheSize)
	}

	if config.ExperimentCacheSize != 0 {
		experimentCache = lru.New(config.ExperimentCacheSize)
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
func saveExperiment2Cache(distinctId string, experiment []beans.Experiment, timeout time.Duration) {
	experimentLock.Lock()
	defer experimentLock.Unlock()
	experimentCache.Add(distinctId, experiment)
	// 进行清理缓存
	removeCache(distinctId, func(id string) {
		experimentCache.Remove(id)
	}, timeout)
}

func castValue(defaultValue interface{}, variables beans.Variables) (bool, interface{}) {
	var defaultType = reflect.TypeOf(defaultValue)
	if variables.Type == "STRING" && "string" == defaultType.String() {
		return true, variables.Value
	} else if variables.Type == "INTEGER" && "int" == defaultType.String() {
		return true, strconv.Atoi(variables.Value)
	} else if variables.Type == "INTEGER" && "int8" == defaultType.String() {
		return true, strconv.ParseInt(variables.Value, 10, 8)
	} else if variables.Type == "INTEGER" && "int16" == defaultType.String() {
		return true, strconv.ParseInt(variables.Value, 10, 16)
	} else if variables.Type == "INTEGER" && "int32" == defaultType.String() {
		return true, strconv.ParseInt(variables.Value, 10, 32)
	} else if variables.Type == "INTEGER" && "int64" == defaultType.String() {
		return true, strconv.ParseInt(variables.Value, 10, 64)
	} else if variables.Type == "BOOLEAN" && "bool" == defaultType.String() {
		return true, strconv.ParseBool(variables.Value)
	}
	return false, nil
}

// 拼接网络请求参数
func buildRequestParam(distinctId string, isLoginId bool, requestParam beans.RequestParam) map[string]interface{} {
	var params = make(map[string]interface{})
	if isLoginId {
		params["login_id"] = distinctId
	} else {
		params["anonymous_id"] = distinctId
	}

	params["abtest_lib_version"] = SDK_VERSION
	params["platform"] = LIB_NAME
	params["properties"] = requestParam.Properties
	return params
}

// 清理缓存
func removeCache(distinctId string, removeCache func(id string), timeout time.Duration) {
	go func() {
		var d time.Duration
		if timeout == 0 {
			d = 24 * time.Hour
		} else {
			d = timeout * time.Minute
		}
		t := time.NewTicker(d)
		defer t.Stop()
		<-t.C
		removeCache(distinctId)
	}()
}
