package cache

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"learning_go/webook/internal/repository/cache/redismocks"
	"testing"
)

func TestRedisCodeCache_Set(t *testing.T) {
	testCases := []struct {
		name    string
		mock    func(ctrl *gomock.Controller) redis.Cmdable
		Biz     string
		code    string
		Phone   string
		wantErr error
	}{
		{
			name:  "发送成功",
			Biz:   "login",
			Phone: "177xx",
			code:  "123",
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				client := redismocks.NewMockCmdable(ctrl)
				//这里ctrl只能控制接口并调用接口的方法，非接口没法调用，.Int()方法不是接口所有的，因此不能直接调用
				//因此这里的返回值cmd只能自己构造
				cmd := redis.NewCmd(context.Background())
				cmd.SetVal(int64(0))
				//set只返回一个值
				client.EXPECT().Eval(gomock.Any(), LuaCodeSet, []string{"phone_code:login:177xx"}, "123").Return(cmd)
				return client
			},
			wantErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			cc := NewCodeCache(tc.mock(ctrl))
			err := cc.Set(context.Background(), tc.Biz, tc.Phone, tc.code)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}

func TestLuaCode(t *testing.T) {
	t.Log(LuaCodeSet) //直接把文件内容嵌入成字符串了
}
