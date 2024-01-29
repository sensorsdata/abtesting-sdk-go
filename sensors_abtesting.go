package sensorsabtest

import (
	"errors"
	"github.com/sensorsdata/abtesting-sdk-go/beans"
	"github.com/sensorsdata/abtesting-sdk-go/utils"
	"github.com/sensorsdata/sa-sdk-go"
)

const (
	SDK_VERSION = "0.1.3"
	LIB_NAME    = "Golang"
)

type SensorsABTest struct {
	config           beans.ABTestConfig
	sensorsAnalytics sensorsanalytics.SensorsAnalytics
}

func InitSensorsABTest(abConfig beans.ABTestConfig) (error, SensorsABTest) {
	err, copyConfig := initConfig(abConfig)
	return err, SensorsABTest{
		config:           copyConfig,
		sensorsAnalytics: abConfig.SensorsAnalytics,
	}
}

/*
拉取最新试验计划
*/
func (sensors *SensorsABTest) AsyncFetchABTest(distinctId string, isLoginId bool, requestParam beans.RequestParam) (error, beans.Experiment) {
	err := checkId(distinctId)
	if err == nil {
		err = checkRequestParams(requestParam)
	}
	if err != nil {
		return err, beans.Experiment{
			Result: requestParam.DefaultValue,
		}
	}

	err, experiment := loadExperimentFromNetwork(sensors, distinctId, isLoginId, requestParam, requestParam.EnableAutoTrackABEvent)

	if err != nil {
		return err, beans.Experiment{
			Result: requestParam.DefaultValue,
		}
	}

	return nil, experiment
}

/*
优先从缓存获取试验变量，如果缓存没有则从网络拉取
*/
func (sensors *SensorsABTest) FastFetchABTest(distinctId string, isLoginId bool, requestParam beans.RequestParam) (error, beans.Experiment) {
	err := checkId(distinctId)
	if err == nil {
		err = checkRequestParams(requestParam)
	}
	if err != nil {
		return err, beans.Experiment{
			Result: requestParam.DefaultValue,
		}
	}

	err, experiment := loadExperimentFromCache(sensors, distinctId, isLoginId, requestParam, requestParam.EnableAutoTrackABEvent)

	if err != nil {
		return err, beans.Experiment{
			Result: requestParam.DefaultValue,
		}

	}

	return nil, experiment
}

func (sensors *SensorsABTest) TrackABTestTrigger(experiment beans.Experiment, property map[string]interface{}) error {
	err := checkId(experiment.DistinctId)
	if err != nil {
		return err
	}
	return sensors.TrackABTestTriggerWithCustomId(experiment, nil, property)
}

func (sensors *SensorsABTest) TrackABTestTriggerWithCustomId(experiment beans.Experiment, customId map[string]interface{}, property map[string]interface{}) error {
	err := checkId(experiment.DistinctId)
	if err != nil {
		return err
	}
	trackABTestEventOuter(experiment.DistinctId, experiment.IsLoginId, experiment, sensors, property, customId)
	return nil
}

// 检查请求参数是否合法
func checkRequestParams(param beans.RequestParam) error {
	if param.ParamName == "" {
		return errors.New("RequestParam.ParamName must not be empty")
	}

	if param.DefaultValue == nil {
		return errors.New("RequestParam.DefaultValue must not be nil")
	}

	var err error
	// 检查自定义属性
	if param.Properties != nil && len(param.Properties) > 0 {
		err = utils.CheckProperty(param.Properties)
	}

	// 检查自定义主体
	if param.CustomIDs != nil && len(param.CustomIDs) > 0 {
		err = utils.CheckCustomIds(param.CustomIDs)
	}
	return err
}

func checkId(id string) error {
	if id == "" {
		return errors.New("DistinctId must not be empty")
	}
	return nil
}

func initConfig(abConfig beans.ABTestConfig) (error, beans.ABTestConfig) {
	if abConfig.APIUrl == "" {
		return errors.New("APIUrl must not be null or empty"), abConfig
	}

	var config = beans.ABTestConfig{}
	if abConfig.ExperimentCacheSize <= 0 {
		config.ExperimentCacheSize = 4096
	} else {
		config.ExperimentCacheSize = abConfig.ExperimentCacheSize
	}

	if abConfig.EventCacheSize <= 0 {
		config.EventCacheSize = 4096
	} else {
		config.EventCacheSize = abConfig.EventCacheSize
	}

	if abConfig.ExperimentCacheTime <= 0 || abConfig.ExperimentCacheTime > 24*60 {
		config.ExperimentCacheTime = 24 * 60
	} else {
		config.ExperimentCacheTime = abConfig.ExperimentCacheTime
	}

	if abConfig.EventCacheTime <= 0 || abConfig.EventCacheTime > 24*60 {
		config.EventCacheTime = 24 * 60
	} else {
		config.EventCacheTime = abConfig.EventCacheTime
	}

	config.SensorsAnalytics = abConfig.SensorsAnalytics
	config.EnableEventCache = abConfig.EnableEventCache
	config.EnableRecordRequestCostTime = abConfig.EnableRecordRequestCostTime
	config.APIUrl = abConfig.APIUrl
	initCache(config)
	utils.InitTransport(getHTTPTransPortParam(abConfig))
	return nil, config
}

func getHTTPTransPortParam(abConfig beans.ABTestConfig) beans.HTTPTransportParam {
	param := beans.HTTPTransportParam{}
	if abConfig.HTTPTransportParam.MaxIdleConnsPerHost <= 0 {
		param.MaxIdleConnsPerHost = 5
	} else {
		param.MaxIdleConnsPerHost = abConfig.HTTPTransportParam.MaxIdleConnsPerHost
	}
	if abConfig.HTTPTransportParam.MaxIdleConns <= 0 {
		param.MaxIdleConns = 20
	} else {
		param.MaxIdleConns = abConfig.HTTPTransportParam.MaxIdleConns
	}

	if abConfig.HTTPTransportParam.MaxConnsPerHost <= 0 {
		param.MaxConnsPerHost = 200
	} else {
		param.MaxConnsPerHost = abConfig.HTTPTransportParam.MaxConnsPerHost
	}

	if abConfig.HTTPTransportParam.IdleConnTimeoutMilliSeconds <= 0 {
		param.IdleConnTimeoutMilliSeconds = 30 * 1000
	} else {
		param.IdleConnTimeoutMilliSeconds = abConfig.HTTPTransportParam.IdleConnTimeoutMilliSeconds
	}

	if abConfig.HTTPTransportParam.DialTimeoutMilliSeconds <= 0 {
		param.DialTimeoutMilliSeconds = 30 * 1000
	} else {
		param.DialTimeoutMilliSeconds = abConfig.HTTPTransportParam.DialTimeoutMilliSeconds
	}

	if abConfig.HTTPTransportParam.DialKeepAliveMilliSeconds <= 0 {
		param.DialKeepAliveMilliSeconds = 30 * 1000
	} else {
		param.DialKeepAliveMilliSeconds = abConfig.HTTPTransportParam.DialKeepAliveMilliSeconds
	}
	return param
}
