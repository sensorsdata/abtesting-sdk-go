package sensorsabtest

import (
	"errors"
	"github.com/sensorsdata/abtesting-sdk-go/beans"
	"github.com/sensorsdata/sa-sdk-go"
)

const (
	SDK_VERSION = "0.0.1"
	LIB_NAME    = "Golang"
)

type SensorsABTest struct {
	config           beans.ABTestConfig
	sensorsAnalytics sensorsanalytics.SensorsAnalytics
}

func InitSensorsABTest(abConfig beans.ABTestConfig) SensorsABTest {
	return SensorsABTest{
		config:           initConfig(abConfig),
		sensorsAnalytics: abConfig.SensorsAnalytics,
	}
}

/*
	拉取最新试验计划
*/
func (sensors *SensorsABTest) AsyncFetchABTest(distinctId string, isLoginId bool, requestParam beans.RequestParam,
	defaultValue interface{}) (error error, variable interface{}, experiment beans.Experiment) {
	_, err := checkRequestParams(distinctId)
	if err != nil {
		return err, nil, beans.Experiment{}
	}

	err, variable, exper := loadExperimentFromNetwork(sensors, distinctId, isLoginId, requestParam, defaultValue, requestParam.EnableAutoTrackABEvent)

	if err != nil {
		return err, defaultValue, beans.Experiment{}
	}

	if requestParam.EnableAutoTrackABEvent {
		return nil, variable, exper
	} else {
		return nil, variable, beans.Experiment{}
	}
}

/*
	优先从缓存获取试验变量，如果缓存没有则从网络拉取
*/
func (sensors *SensorsABTest) FastFetchABTest(distinctId string, isLoginId bool, requestParam beans.RequestParam,
	defaultValue interface{}) (error error, variable interface{}, experiment beans.Experiment) {
	_, err := checkRequestParams(distinctId)
	if err != nil {
		return err, nil, beans.Experiment{}
	}

	err, variable, exper := loadExperimentFromCache(sensors, distinctId, isLoginId, requestParam, defaultValue, requestParam.EnableAutoTrackABEvent)

	if err != nil {
		return err, defaultValue, beans.Experiment{}
	}

	if requestParam.EnableAutoTrackABEvent {
		return nil, variable, exper
	} else {
		return nil, variable, beans.Experiment{}
	}
}

func (sensors *SensorsABTest) TrackABTestTrigger(distinctId string, isLoginId bool, experiment beans.Experiment, property map[string]interface{}) error {
	_, err := checkRequestParams(distinctId)
	if err != nil {
		return err
	}
	trackABTestEvent(distinctId, isLoginId, experiment, sensors, property)
	return nil
}

// 检查请求参数是否合法
func checkRequestParams(distinctId string) (bool, error) {
	if distinctId == "" {
		return false, errors.New("DistinctId must not be empty")
	}
	return true, nil
}

func initConfig(abConfig beans.ABTestConfig) beans.ABTestConfig {
	var config = beans.ABTestConfig{}
	if abConfig.ExperimentCacheSize <= 0 {
		config.ExperimentCacheSize = 4096
	}

	if abConfig.EventCacheSize <= 0 {
		config.EventCacheSize = 4096
	}

	if abConfig.ExperimentCacheTime <= 0 {
		config.ExperimentCacheTime = 24 * 60 * 60
	}

	if abConfig.EventCacheTime <= 0 {
		config.EventCacheTime = 24 * 60 * 60
	}

	if abConfig.Timeout <= 0 {
		config.Timeout = 3
	}

	config.SensorsAnalytics = abConfig.SensorsAnalytics
	config.EnableEventCache = abConfig.EnableEventCache
	config.APIUrl = abConfig.APIUrl
	initCache(config)
	return config
}
