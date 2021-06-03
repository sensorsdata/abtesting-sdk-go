package beans

import "time"

type ABConfig struct {
	/*
		试验缓存时间
	*/
	ExperimentCacheTime int
	/*
		试验总缓存用户量限制
	*/
	ExperimentCacheSize int

	/*
		$ABTestTrigger 事件单用户缓存配置
	*/
	EventCacheTime int
	/*
		$ABTestTrigger 事件总缓存用户量限制
	*/
	EventCacheSize int

	/*
		开启 A/B 事件缓存
	*/
	EnableEventCache bool

	/*
		API 地址
	*/
	ServerAPIUrl string

	/**
	网络请求超时时间
	*/
	Timeout time.Duration
}