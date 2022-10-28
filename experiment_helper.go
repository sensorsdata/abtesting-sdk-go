package sensorsabtest

import (
	"errors"
	"fmt"
	utils2 "github.com/sensorsdata/sa-sdk-go/utils"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/sensorsdata/abtesting-sdk-go/beans"
	"github.com/sensorsdata/abtesting-sdk-go/utils"
	"github.com/sensorsdata/abtesting-sdk-go/utils/lru"
)

// 用户的试验缓存
var experimentCache = lru.New(4096)
var experimentLock = sync.Mutex{}
var experimentTime = lru.New(4096)

// $ABTestTrigger 事件缓存
var eventsTime = lru.New(4096)
var eventsLock = sync.Mutex{}

// 插件版本号标记位
var isFirstEvent = true

// 埋点事件上次触发的时间
var lastTimeEvent string

func loadExperimentFromNetwork(sensors *SensorsABTest, distinctId string, isLoginId bool, requestParam beans.RequestParam, isTrack bool) (error, beans.Experiment) {
	if requestParam.TimeoutMilliseconds <= 0 {
		requestParam.TimeoutMilliseconds = 3 * 1000
	}
	experiments, err := requestExperimentOnNetwork(sensors.config.APIUrl, distinctId, isLoginId, requestParam)
	if err != nil {
		return err, beans.Experiment{
			Result: requestParam.DefaultValue,
		}
	}

	experiment := filterExperiment(requestParam, experiments)
	if experiment.AbtestExperimentId != "" {
		if isTrack {
			trackABTestEvent(distinctId, isLoginId, experiment, sensors, nil, requestParam.CustomIDs)
		}
		// 回调试验变量给客户
		experiment.DistinctId = distinctId
		experiment.IsLoginId = isLoginId
		return nil, experiment
	}

	return nil, beans.Experiment{
		DistinctId: distinctId,
		IsLoginId:  isLoginId,
		Result:     requestParam.DefaultValue,
	}
}

func loadExperimentFromCache(sensors *SensorsABTest, distinctId string, isLoginId bool, requestParam beans.RequestParam, isTrack bool) (error, beans.Experiment) {
	var experiment beans.Experiment
	var isRequestNetwork = false
	idKey := getExperimentKey(distinctId, requestParam.CustomIDs, isLoginId)
	if isExperimentExpired(idKey, sensors.config.ExperimentCacheTime) {
		idKey := getExperimentKey(distinctId, requestParam.CustomIDs, isLoginId)
		// 进行清理缓存
		experimentCache.Remove(idKey)
		isRequestNetwork = true
	} else {
		experiments, ok := loadExperimentCache(idKey)
		if experiments != nil || ok {
			experiment = filterExperiment(requestParam, experiments.([]beans.Experiment))
		}
		if experiments == nil || !ok || experiment.AbtestExperimentId == "" {
			isRequestNetwork = true
		}
	}

	if isRequestNetwork {
		// 从网络请求试验
		experiments, err := requestExperimentOnNetwork(sensors.config.APIUrl, distinctId, isLoginId, requestParam)
		if err != nil {
			return err, beans.Experiment{
				Result: requestParam.DefaultValue,
			}
		}

		// 缓存试验
		saveExperiment2Cache(idKey, experiments)
		// 筛选试验
		experiment = filterExperiment(requestParam, experiments)
	}

	if isTrack && experiment.AbtestExperimentId != "" {
		trackABTestEvent(distinctId, isLoginId, experiment, sensors, nil, requestParam.CustomIDs)
	}
	experiment.DistinctId = distinctId
	experiment.IsLoginId = isLoginId
	return nil, experiment
}

// 从网络加载试验
func requestExperimentOnNetwork(apiUrl string, distinctId string, isLoginId bool, requestParam beans.RequestParam) ([]beans.Experiment, error) {
	if requestParam.TimeoutMilliseconds <= 0 {
		requestParam.TimeoutMilliseconds = 3 * 1000
	}
	return utils.RequestExperiment(apiUrl, buildRequestParam(distinctId, isLoginId, requestParam), time.Duration(requestParam.TimeoutMilliseconds)*time.Millisecond)
}

// 筛选试验
func filterExperiment(requestParam beans.RequestParam, experiments []beans.Experiment) beans.Experiment {
	var experimentParam = requestParam.ParamName
	// 遍历试验
	for _, experiment := range experiments {
		// 遍历试验变量
		for _, variable := range experiment.VariableList {
			if experimentParam == variable.Name {
				value, err := castValue(requestParam.DefaultValue, variable)
				if err == nil {
					experiment.Result = value
					return experiment
				}
			}
		}
	}

	return beans.Experiment{
		Result: requestParam.DefaultValue,
	}
}

func trackABTestEvent(distinctId string, isLoginId bool, experiment beans.Experiment, sensors *SensorsABTest, properties map[string]interface{}, customIDs map[string]interface{}) {
	if sensors.config.SensorsAnalytics.C == nil {
		return
	}
	// 是白名单，则不触发 $ABTestTrigger 事件
	if experiment.IsWhiteList {
		return
	}
	idEvent := getEventKey(distinctId, customIDs, experiment)
	// 如果在缓存中，则不触发 $ABTestTrigger 事件
	ok := isEventExistAndNotExpired(idEvent, sensors.config.EventCacheTime)
	if ok {
		return
	}

	saveEvent2Cache(idEvent, sensors)
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
		eventsTime = lru.New(config.EventCacheSize)
	}

	if config.ExperimentCacheSize != 0 {
		experimentCache = lru.New(config.ExperimentCacheSize)
		experimentTime = lru.New(config.ExperimentCacheSize)
	}
}

// 从缓存读取 $ABTestTrigger
func isEventExistAndNotExpired(idEvent string, timeout time.Duration) bool {
	eventsLock.Lock()
	defer eventsLock.Unlock()
	lastTime, ok := eventsTime.Get(idEvent)
	if ok {
		return (utils2.NowMs() - lastTime.(int64)) < int64(timeout*time.Minute/time.Millisecond)
	}

	return false
}

// 保存 $ABTestTrigger 到缓存中
func saveEvent2Cache(idEvent string, sensors *SensorsABTest) {
	// 缓存 $ABTestTrigger 事件
	if sensors.config.EnableEventCache {
		eventsLock.Lock()
		defer eventsLock.Unlock()
		eventsTime.Remove(idEvent)
		eventsTime.Add(idEvent, utils2.NowMs())
	}
}

// 从缓存读取试验
func loadExperimentCache(idKey string) (interface{}, bool) {
	experimentLock.Lock()
	defer experimentLock.Unlock()
	return experimentCache.Get(idKey)
}

// 保存试验到缓存
func saveExperiment2Cache(idKey string, experiments []beans.Experiment) {
	experimentLock.Lock()
	defer experimentLock.Unlock()
	experimentCache.Add(idKey, experiments)
	experimentTime.Add(idKey, utils2.NowMs())
}

func castValue(defaultValue interface{}, variables beans.Variables) (interface{}, error) {
	var defaultType = reflect.TypeOf(defaultValue)
	if (variables.Type == "STRING" || variables.Type == "JSON") && "string" == defaultType.String() {
		return variables.Value, nil
	} else if variables.Type == "INTEGER" && "int" == defaultType.String() {
		return strconv.Atoi(variables.Value)
	} else if variables.Type == "INTEGER" && "int8" == defaultType.String() {
		return strconv.ParseInt(variables.Value, 10, 8)
	} else if variables.Type == "INTEGER" && "int16" == defaultType.String() {
		return strconv.ParseInt(variables.Value, 10, 16)
	} else if variables.Type == "INTEGER" && "int32" == defaultType.String() {
		return strconv.ParseInt(variables.Value, 10, 32)
	} else if variables.Type == "INTEGER" && "int64" == defaultType.String() {
		return strconv.ParseInt(variables.Value, 10, 64)
	} else if variables.Type == "BOOLEAN" && "bool" == defaultType.String() {
		return strconv.ParseBool(variables.Value)
	}
	return defaultValue, errors.New("castValue No Type Found")
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
	if requestParam.Properties != nil && len(requestParam.Properties) > 0 {
		params["custom_properties"] = requestParam.Properties
	}
	if requestParam.CustomIDs != nil && len(requestParam.CustomIDs) > 0 {
		params["custom_ids"] = requestParam.CustomIDs
	}

	return params
}

// 拼接缓存唯一标识
func getEventKey(distinctId string, customIds map[string]interface{}, experiment beans.Experiment) string {
	return distinctId + "$" + utils.MapToJson(customIds) + "$" + experiment.AbtestExperimentId
}

func getExperimentKey(distinctId string, customIds map[string]interface{}, isLoginId bool) string {
	return distinctId + "$" + utils.MapToJson(customIds) + "$" + strconv.FormatBool(isLoginId)
}

func isExperimentExpired(idKey string, timeout time.Duration) bool {
	lastTime, ok := experimentTime.Get(idKey)
	if ok {
		return (utils2.NowMs() - lastTime.(int64)) > int64(timeout*time.Minute/time.Millisecond)
	}
	return false
}
