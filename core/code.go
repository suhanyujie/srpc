package core

import "io"

// 编解码抽象 code interface
type CodeIf interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header, interface{}) error
}

type NewCodeFunc func(conn io.ReadWriteCloser) CodeIf

type Type string

const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json"
)

type NewCodeFuncMap map[Type]NewCodeFunc

var CodeFuncMap NewCodeFuncMap

func init() {
	CodeFuncMap = make(map[Type]NewCodeFunc)
	CodeFuncMap[GobType] = NewGobCode
}
