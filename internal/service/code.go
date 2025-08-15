package service

import (
	"context"
	"fmt"
	"learning_go/webook/internal/repository"
	"learning_go/webook/internal/service/sms"
	"math/rand/v2"
)

const codeTplId = "1877556"

type CodeService struct {
	svc  sms.Service //面向接口，依赖注入
	repo *repository.CodeRepository
}

func NewCodeService(svc sms.Service, repo *repository.CodeRepository) *CodeService {
	return &CodeService{svc: svc, repo: repo}
}

// biz用来区分业务，例如修改密码的验证码和登录的验证码要区别
func (cs *CodeService) Send(ctx context.Context, biz string, phone string) error {
	//考虑在服务层生成验证码
	code := cs.generateCode()
	//在redis中设置验证码
	err := cs.repo.Store(ctx, biz, phone, code)
	if err != nil {
		return err
	}
	//别忘了发送短信
	err = cs.svc.Send(ctx, codeTplId, []string{code}, phone)
	if err != nil {
		//这里redis已经存储了，但是短信服务出错
		//如果是短信服务的超时err，这里要重试？
		//如果要重试，不能放在这里做，而是要在smssvc里实现重试功能

	}
	return err
}

// 使用两个返回值，bool用来区分业务的对错（码的对错），error区分系统错误
// 也可以直接一个error，后续再区分
func (cs *CodeService) Verify(ctx context.Context, biz string, phone string, inputCode string) error {
	return cs.repo.Verify(ctx, biz, phone, inputCode)
}

func (cs *CodeService) generateCode() string {
	num := rand.IntN(1000000)       //6位数，最大999999
	return fmt.Sprintf("%06d", num) //加上前导0
}
