package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"learning_go/webook/internal/domain"
	"time"
)

var ErrorKeyNotExist = redis.Nil

type RedisUserCache struct {
	//面向接口编程，结构体的接口依赖由外部注入
	//这样，这里就可以传入单机的redis，也可以传入cluster集群的redis
	client     redis.Cmdable
	expiration time.Duration
}

type UserCache interface {
	Get(ctx context.Context, id int64) (domain.User, error)
	Set(ctx context.Context, u domain.User) error
}

// NewUserCache
// 面向接口编程与依赖注入 “三把斧”
// A用到了B，B一定是接口
// A用到了B，B一定是A的字段
// A用到了B，A绝对不初始化B，而是外面注入
func NewUserCache(client redis.Cmdable) *RedisUserCache {
	return &RedisUserCache{client: client, expiration: time.Minute * 15}
}

// 如果没有数据，返回一个特定error，因为业务层那边要区分是没数据还是redis出错了
func (uc *RedisUserCache) Get(ctx context.Context, id int64) (domain.User, error) {
	val, err := uc.client.Get(ctx, uc.key(id)).Bytes()
	if err != nil {
		return domain.User{}, err
	}
	var user domain.User
	err = json.Unmarshal(val, &user)
	return user, err
}

func (uc *RedisUserCache) Set(ctx context.Context, u domain.User) error {
	val, err := json.Marshal(u) //json序列化，用于把一个结构体对象转为json格式的数据 FirstPageKey:value
	if err != nil {
		return err
	}
	//ctx是一个请求的上下文，redis传入ctx可以监听请求的取消信号、利用ctx在不同逻辑层传递信息等等
	//这里不能直接传结构体，可以传入json序列化后的byte[]字节数据，后续再通过json反序列化出来即可
	return uc.client.Set(ctx, uc.key(u.Id), val, uc.expiration).Err()
}

func (uc *RedisUserCache) key(id int64) string {
	return fmt.Sprintf("user_info_%d", id)
}
