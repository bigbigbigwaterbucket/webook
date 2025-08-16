package cache

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
)

// go嵌入外部代码到string字符串里

//go:embed lua/Set_code.lua
var LuaCodeSet string

//go:embed lua/Verify_code.lua
var LuaCodeVerify string

// 对于特殊的error，都需要定义，方便后续服务处理，并且在哪出错，在哪定义
var ErrorCodeSendTooMany = errors.New("发送验证码太频繁")
var ErrorCodeVerifyTooManyTimes = errors.New("验证次数过多")
var ErrorCodeNotRight = errors.New("验证码错误")

type RedisCodeCache struct {
	client redis.Cmdable
}

type CodeCache interface {
	Set(ctx context.Context, biz, phone, code string) error
	Verify(ctx context.Context, biz, phone, inputCode string) error
}

func NewCodeCache(client redis.Cmdable) *RedisCodeCache {
	return &RedisCodeCache{client: client}
}

func (cc *RedisCodeCache) Set(ctx context.Context, biz, phone, code string) error {
	//在go中调用redis的lua脚本，实现原子操作
	//ctx：上下文，用于控制超时或取消
	//script：Lua 脚本字符串
	//keys：Lua 脚本中使用的 KEYS 参数列表
	//args：Lua 脚本中使用的 ARGV 参数列表
	res, err := cc.client.Eval(ctx, LuaCodeSet, []string{cc.key(phone, biz)}, code).Int()
	if err != nil {
		//也是系统错误
		return err
	}
	switch res {
	case 0:
		return nil
	case -1:
		return ErrorCodeSendTooMany
	default:
		//同事没设置expire time
		return errors.New("系统错误")
	}
}

func (cc *RedisCodeCache) Verify(ctx context.Context, biz, phone, inputCode string) error {
	res, err := cc.client.Eval(ctx, LuaCodeVerify, []string{cc.key(phone, biz)}, inputCode).Int()
	if err != nil {
		return err
	}
	switch res {
	case 0:
		return nil
	case -1:
		return ErrorCodeVerifyTooManyTimes
	case -2:
		return ErrorCodeNotRight
	}
	return errors.New("系统错误")
}
func (cc *RedisCodeCache) key(phone, biz string) string {
	return fmt.Sprintf("phone_code:%s:%s", biz, phone)
}
