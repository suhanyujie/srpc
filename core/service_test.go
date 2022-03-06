package core

import (
	"fmt"
	"reflect"
	"testing"
)

type Foo int

type Args struct {
	Num, Num2 int
}

func (f Foo) Sum(arg Args, reply *int) error {
	*reply = arg.Num + arg.Num2
	return nil
}

func (f Foo) sum(args Args, reply *int) error {
	*reply = args.Num2 + args.Num
	return nil
}

func _assert(condIsOk bool, msg string, v ...interface{}) {
	if !condIsOk {
		panic(fmt.Sprintf("assert failed: "+msg, v...))
	}
}

// 测试实体（对象）及其方法的注册是否正确
func TestNewService(t *testing.T) {
	var foo Foo
	s := newService(&foo)
	_assert(len(s.method) == 1, "wrong service method, expect 1, but got %d", len(s.method))
	mType := s.method["Sum"]
	_assert(mType != nil, "wrong Method, Sum shouldn't nil")
}

func TestMehtodCall(t *testing.T) {
	var foo Foo
	s := newService(&foo)

	mType := s.method["Sum"]
	argv := mType.newArgv()
	replyv := mType.newReplyv()
	argv.Set(reflect.ValueOf(Args{
		Num:  1,
		Num2: 17,
	}))
	err := s.call(mType, argv, replyv)
	isOk := err == nil && *replyv.Interface().(*int) == 18 && mType.NumCalls() == 1
	_assert(isOk, "failed to call Foo.Sum")
}
