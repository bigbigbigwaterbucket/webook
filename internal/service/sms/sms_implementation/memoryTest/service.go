package memoryTest

import (
	"context"
	"fmt"
)

type MemService struct {
}

func NewMemService() *MemService {
	return &MemService{}
}

// 模拟发短信过程，测试用
func (s *MemService) Send(ctx context.Context, tpl string, args []string, number ...string) error {
	fmt.Println(args)
	return nil
}
