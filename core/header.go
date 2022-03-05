package core

// 主要类型的定义

type Header struct {
	// 请求方法。
	Method string `json:"method"`
	// 请求序列号
	TraceId string
	// 存储一些元信息，用于一些扩展功能
	MetaData map[string]interface{}
	// 错误类型
	Error string
}
