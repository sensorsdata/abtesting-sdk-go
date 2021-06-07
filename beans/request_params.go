package beans

type RequestParam struct {
	// 试验变量名
	ParamName string

	// HTTP 请求参数
	Properties map[string]interface{}
}
