package sms

import "context"

// 可变参数...相比切片区别在于更随意，可以直接传字符串，还可以传0个字符串，不是一定要初始化一个切片
type Service interface {
	Send(ctx context.Context, tpl string, args []string, number ...string) error
}
