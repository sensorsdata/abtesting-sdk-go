package beans

type RequestParam struct {
	// 试验变量名
	ParamName string

	// 默认值
	DefaultValue interface{}

	// HTTP 请求参数
	Properties map[string]interface{}

	// 自定义分流主体
	CustomIDs map[string]interface{}

	// 网络请求超时时间，单位 ms，默认 3s
	TimeoutMilliseconds int

	// 是否自动采集 A/B Testing 埋点事件
	EnableAutoTrackABEvent bool
}
