package tencent

import (
	"context"
	"fmt"
	"github.com/ecodeclub/ekit"
	"github.com/ecodeclub/ekit/slice"
	sms "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111"
)

type TencentService struct {
	appId    *string
	signName *string
	client   *sms.Client
}

func NewTencentService(c *sms.Client, appId string,
	signName string) *TencentService {
	return &TencentService{
		client:   c,
		appId:    ekit.ToPtr[string](appId), //也方便设置常量，go会分析函数局部变量逃逸
		signName: ekit.ToPtr[string](signName),
	}
}

func (s *TencentService) Send(ctx context.Context, tpl string, args []string, numbers ...string) error {
	req := sms.NewSendSmsRequest()
	req.PhoneNumberSet = toStringPtrSlice(numbers)
	req.SmsSdkAppId = s.appId
	// ctx 继续往下传
	req.SetContext(ctx)
	req.TemplateParamSet = toStringPtrSlice(args)
	req.TemplateId = ekit.ToPtr[string](tpl)
	req.SignName = s.signName
	resp, err := s.client.SendSms(req)
	if err != nil {
		return err
	}
	for _, status := range resp.Response.SendStatusSet {
		if status.Code == nil || *(status.Code) != "Ok" {
			return fmt.Errorf("发送失败，code: %s, 原因：%s",
				*status.Code, *status.Message)
		}
	}
	return nil
}

func toStringPtrSlice(src []string) []*string {
	//deng ming的数据结构工具库，map实现遍历切片的每个元素，并应用自定义函数参数到每个元素上
	//返回应用后的结果切片
	return slice.Map[string, *string](src, func(idx int, src string) *string {
		return &src
	})
}
