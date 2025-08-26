package web

import (
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"learning_go/webook/internal/domain"
	"learning_go/webook/internal/service"
	svcmocks "learning_go/webook/internal/service/mocks"
	"learning_go/webook/internal/web/ijwt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestArticleHandler_Publish(t *testing.T) {
	testCases := []struct {
		name     string
		mock     func(ctrl *gomock.Controller) service.ArticleService
		reqBody  string
		wantCode int
		wantRes  Result
	}{
		{
			name: "发布成功",
			//传指针还有可能是结构体内部有指针类型，值传递会赋值值
			mock: func(ctrl *gomock.Controller) service.ArticleService {
				svc := svcmocks.NewMockArticleService(ctrl)
				svc.EXPECT().Publish(gomock.Any(), domain.Article{Id: 123, Title: "我的标题", Content: "我的内容", Author: domain.Author{Id: 666}}).Return(nil)
				return svc
			},
			//json格式数据最后不能加逗号, !!
			reqBody: `
{
	"Id":123,
	"Title":"我的标题",
	"Content":"我的内容"
}
			`,
			wantCode: http.StatusOK,
			wantRes:  Result{Msg: "OK", Data: float64(123)},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			svc := tc.mock(ctrl)
			articleHandler := ArticleHandler{svc: svc}
			server := gin.Default()
			//先注册中间件，再注册路由！
			server.Use(func(context *gin.Context) {
				context.Set("user", ijwt.UserClaims{Uid: 666})
			})
			articleHandler.RegisterRouter(server)
			//http通信传输的数据都是字节流[]byte，reader/writer多了缓冲区的字节读/写器等
			req, err := http.NewRequest(http.MethodPost, "/articles/publish", bytes.NewBuffer([]byte(tc.reqBody)))
			assert.NoError(t, err)
			//这里请求头别忘记设置，gin的bind方法是通过请求头解析bytes数据的
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			server.ServeHTTP(resp, req)
			assert.Equal(t, tc.wantCode, resp.Code)
			var res Result
			err = json.Unmarshal(resp.Body.Bytes(), &res)
			assert.NoError(t, err)
			assert.Equal(t, tc.wantRes, res)
		})
	}
}
