package beans

type RequestParam struct {
	// 试验变量名
	ExperimentParam string

	// HTTP 请求参数
	HttpRequestPrams map[string]interface{}
}
