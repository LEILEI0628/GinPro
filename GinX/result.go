package ginx

type Result struct {
	// 业务状态码
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}
