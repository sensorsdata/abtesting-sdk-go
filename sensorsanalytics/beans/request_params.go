package beans

type RequestParam struct {
	// 试验变量名
	ExperimentParam string
	// 匿名 ID
	AnonymousId string
	// 登录 ID
	LoginId string

	// HTTP 请求参数
	HttpRequestPrams map[string]interface{}
}
