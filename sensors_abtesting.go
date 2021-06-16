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
func (sensors *SensorsABTest) AsyncFetchABTest(distinctId string, isLoginId bool, requestParam beans.RequestParam) (error error, variable interface{}, experiment beans.Experiment) {
	err := checkId(distinctId)
	if err == nil {
		err = checkRequestParams(requestParam)
	}
	if err != nil {
		return err, nil, beans.Experiment{}
	}

	err, variable, exper := loadExperimentFromNetwork(sensors, distinctId, isLoginId, requestParam, requestParam.EnableAutoTrackABEvent)

	if err != nil {
		return err, requestParam.DefaultValue, beans.Experiment{}
	}

	return nil, variable, exper
}

/*
	优先从缓存获取试验变量，如果缓存没有则从网络拉取
*/
func (sensors *SensorsABTest) FastFetchABTest(distinctId string, isLoginId bool, requestParam beans.RequestParam) (error error, variable interface{}, experiment beans.Experiment) {
	err := checkId(distinctId)
	if err == nil {
		err = checkRequestParams(requestParam)
	}
	if err != nil {
		return err, nil, beans.Experiment{}
	}

	err, variable, exper := loadExperimentFromCache(sensors, distinctId, isLoginId, requestParam, requestParam.EnableAutoTrackABEvent)

	if err != nil {
		return err, requestParam.DefaultValue, beans.Experiment{}
	}

	return nil, variable, exper
}

func (sensors *SensorsABTest) TrackABTestTrigger(experiment beans.Experiment, property map[string]interface{}) error {
	err := checkId(experiment.DistinctId)
	if err != nil {
		return err
	}
	trackABTestEvent(experiment.DistinctId, experiment.IsLoginId, experiment, sensors, property)
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

	return nil
}

func checkId(id string) error {
	if id == "" {
		return errors.New("DistinctId must not be empty")
	}
	return nil
}

func initConfig(abConfig beans.ABTestConfig) beans.ABTestConfig {
	var config = beans.ABTestConfig{}
	if abConfig.ExperimentCacheSize <= 0 {
		config.ExperimentCacheSize = 4096
	}

	if abConfig.EventCacheSize <= 0 {
		config.EventCacheSize = 4096
	}

	if abConfig.ExperimentCacheTime <= 0 || abConfig.ExperimentCacheTime > 24*60 {
		config.ExperimentCacheTime = 24 * 60
	}

	if abConfig.EventCacheTime <= 0 || abConfig.EventCacheTime > 24*60 {
		config.EventCacheTime = 24 * 60
	}

	config.SensorsAnalytics = abConfig.SensorsAnalytics
	config.EnableEventCache = abConfig.EnableEventCache
	config.APIUrl = abConfig.APIUrl
	initCache(config)
	return config
}
