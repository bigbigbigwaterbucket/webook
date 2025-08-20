package auth

import (
	"context"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"learning_go/webook/internal/service/sms"
)

type JWTAuthSmsService struct {
	svc sms.Service
	//加密/解密token用的字符串
	key string
}

// 短息服务鉴权是给内部业务用的，这里带tpl的token必须是线下申请的，当作密钥一样通过文档发过去
// 业务方复制粘贴一下就可以了
func (J *JWTAuthSmsService) Send(ctx context.Context, token string, args []string, number ...string) error {
	var smsClaim SmsClaims
	//内部服务鉴权，做的比较简单，能解析成功说明就是对应的业务方
	tokenReal, err := jwt.ParseWithClaims(token, &smsClaim, func(token *jwt.Token) (any, error) {
		//返回给token加密/解密的key
		return J.key, nil
	})
	if err != nil {
		return err
	}
	//解密后的真实token
	if !tokenReal.Valid {
		return errors.New("短信token过期")
	}
	err = J.svc.Send(ctx, smsClaim.tpl, args, number...)
	return err
}

type SmsClaims struct {
	jwt.RegisteredClaims
	tpl string
}
