package memoryTest

import (
	"context"
	"fmt"
)

type Service struct {
}

func NewMemService() *Service {
	return &Service{}
}

// 模拟发短信过程，测试用
func (s *Service) Send(ctx context.Context, tpl string, args []string, number ...string) error {
	fmt.Println(args)
	return nil
}
