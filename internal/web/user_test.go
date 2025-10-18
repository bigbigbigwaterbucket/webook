package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"learning_go/webook/internal/repository/cache"
	"learning_go/webook/internal/service"
	svcmocks "learning_go/webook/internal/service/mocks"
	"learning_go/webook/user/domain"
	service2 "learning_go/webook/user/service"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
)

func TestEncrypt(t *testing.T) {
	password := "hello#world123"
	encrypted, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	err = bcrypt.CompareHashAndPassword(encrypted, []byte(password))
	assert.NoError(t, err)
}

func TestUserHandler_SignUp(t *testing.T) {
	const signupUrl = "/users/signup"
	testCases := []struct {
		name string

		mock func(ctrl *gomock.Controller) (service2.UserService, service.CodeService)

		reqBuilder func(t *testing.T) *http.Request
		wantCode   int
		wantBody   string
	}{
		{
			name: "注册成功",
			//匿名函数，接收一个ctrl，每次测试都接收，返回该次测试你用到的依赖接口
			mock: func(ctrl *gomock.Controller) (service2.UserService, service.CodeService) {
				usersvc := svcmocks.NewMockUserService(ctrl)
				//signup的时候没有用到codesvc，因此预期不应该调用其函数
				codesvc := svcmocks.NewMockCodeService(ctrl)
				//注册成功signup返回值应该是nil
				usersvc.EXPECT().SignUp(gomock.Any(), gomock.Any()).Return(nil)
				return usersvc, codesvc
			},
			//构造每次测试的请求体
			reqBuilder: func(t *testing.T) *http.Request {
				body := bytes.NewBuffer([]byte(`{"email":"123@qq.com","password":"hello@world123","confirmPassword":"hello@world123"}`))
				req, err := http.NewRequest(http.MethodPost, signupUrl, body)
				//这里必须要设置一下，不然gin没法解析内容？
				req.Header.Set("Content-Type", "application/json")
				if err != nil {
					t.Fatal(err)
				}
				return req
			},
			wantCode: 200,
			wantBody: "注册成功",
		},
		{
			name: "json格式错误",
			mock: func(ctrl *gomock.Controller) (service2.UserService, service.CodeService) {
				return nil, nil
			},
			//构造每次测试的请求体
			reqBuilder: func(t *testing.T) *http.Request {
				body := bytes.NewBuffer([]byte(`{"email":"123@qq.com","password":"hello@world123","confirmPass`))
				req, err := http.NewRequest(http.MethodPost, signupUrl, body)
				//这里必须要设置一下，不然gin没法解析内容？
				req.Header.Set("Content-Type", "application/json")
				if err != nil {
					t.Fatal(err)
				}
				return req
			},
			wantCode: 400,
			wantBody: "json格式错误",
		},
		{
			name: "邮箱格式不对",
			//邮箱格式不对时，在web层就应该被拦截，不应该去调用usersvc去执行signup
			mock: func(ctrl *gomock.Controller) (service2.UserService, service.CodeService) {
				usersvc := svcmocks.NewMockUserService(ctrl)
				codesvc := svcmocks.NewMockCodeService(ctrl)
				return usersvc, codesvc
			},
			//构造每次测试的请求体
			reqBuilder: func(t *testing.T) *http.Request {
				body := bytes.NewBuffer([]byte(`{"email":"123m","password":"hello@world123","confirmPassword":"hello@world123"}`))
				req, err := http.NewRequest(http.MethodPost, signupUrl, body)
				//这里必须要设置一下，不然gin没法解析内容？
				req.Header.Set("Content-Type", "application/json")
				if err != nil {
					t.Fatal(err)
				}
				return req
			},
			wantCode: 200,
			wantBody: "邮箱格式错误",
		},
		{
			name: "两次输入密码不同",
			//也会被web层拦截
			mock: func(ctrl *gomock.Controller) (service2.UserService, service.CodeService) {
				return nil, nil
			},
			//构造每次测试的请求体
			reqBuilder: func(t *testing.T) *http.Request {
				body := bytes.NewBuffer([]byte(`{"email":"123@qq.com","password":"hello@world13","confirmPassword":"hello@world123"}`))
				req, err := http.NewRequest(http.MethodPost, signupUrl, body)
				//这里必须要设置一下，不然gin没法解析内容？
				req.Header.Set("Content-Type", "application/json")
				if err != nil {
					t.Fatal(err)
				}
				return req
			},
			wantCode: 200,
			wantBody: "两次输入密码不一致",
		},
		{
			name: "邮箱冲突",
			//也会被web层拦截
			mock: func(ctrl *gomock.Controller) (service2.UserService, service.CodeService) {
				usersvc := svcmocks.NewMockUserService(ctrl)
				usersvc.EXPECT().SignUp(gomock.Any(), gomock.Any()).Return(service2.ErrUserDuplicate)
				return usersvc, nil
			},
			//构造每次测试的请求体
			reqBuilder: func(t *testing.T) *http.Request {
				body := bytes.NewBuffer([]byte(`{"email":"123@qq.com","password":"hello@world123","confirmPassword":"hello@world123"}`))
				req, err := http.NewRequest(http.MethodPost, signupUrl, body)
				//这里必须要设置一下，不然gin没法解析内容？
				req.Header.Set("Content-Type", "application/json")
				if err != nil {
					t.Fatal(err)
				}
				return req
			},
			wantCode: 200,
			wantBody: "邮箱冲突",
		},
		{
			name: "系统异常",
			mock: func(ctrl *gomock.Controller) (service2.UserService, service.CodeService) {
				usersvc := svcmocks.NewMockUserService(ctrl)
				usersvc.EXPECT().SignUp(gomock.Any(), gomock.Any()).Return(errors.New("模拟系统异常"))
				return usersvc, nil
			},
			//构造每次测试的请求体
			reqBuilder: func(t *testing.T) *http.Request {
				body := bytes.NewBuffer([]byte(`{"email":"123@qq.com","password":"hello@world123","confirmPassword":"hello@world123"}`))
				req, err := http.NewRequest(http.MethodPost, signupUrl, body)
				//这里必须要设置一下，不然gin没法解析内容？
				req.Header.Set("Content-Type", "application/json")
				if err != nil {
					t.Fatal(err)
				}
				return req
			},
			wantCode: 200,
			wantBody: "系统错误",
		},
	}
	//测试的时候，mock构建的组件不会实际执行其函数，但是非mock的组件，
	//即测试组件，这里是userhandler的函数是会被实际执行的！
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			//mock解决组件依赖问题
			usersvc, codesvc := tc.mock(ctrl)
			hdl := NewUserHandler(usersvc, codesvc, nil)
			server := gin.Default()
			hdl.RegisterRouter(server)
			//准备请求
			req := tc.reqBuilder(t)
			//记录响应内容
			recoder := httptest.NewRecorder()
			//将请求转化为响应，这是一个处理函数
			server.ServeHTTP(recoder, req)
			assert.Equal(t, tc.wantBody, recoder.Body.String())
			assert.Equal(t, tc.wantCode, recoder.Code)
		})
	}
}

func TestMock(t *testing.T) {
	//创建一个控制mock的控制器
	ctrl := gomock.NewController(t)
	//测试结束后调用finish，mock会验证测试流程是否符合预期
	defer ctrl.Finish()
	usersvc := svcmocks.NewMockUserService(ctrl)
	//写预期调用signup函数，预期传入xxx参数，预期返回xxx错误
	usersvc.EXPECT().SignUp(gomock.Any(), gomock.Any()).Return(errors.New("mock error"))
	//开始实际调用
	// background传入一个空白上下文
	err := usersvc.SignUp(context.Background(), domain.User{Email: "123@qq.com"})
	t.Log(err)
}

func TestUserHandler_LoginSmsCode(t *testing.T) {
	testCases := []struct {
		name       string
		reqBuilder func(t *testing.T) *http.Request
		mock       func(ctrl *gomock.Controller) (service2.UserService, service.CodeService)
		wantRes    Result
	}{
		{
			name: "登录成功",
			mock: func(ctrl *gomock.Controller) (service2.UserService, service.CodeService) {
				us := svcmocks.NewMockUserService(ctrl)
				cs := svcmocks.NewMockCodeService(ctrl)
				cs.EXPECT().Verify(gomock.Any(), "login", gomock.Any(), gomock.Any()).Return(nil)
				us.EXPECT().FindOrCreateByPhone(gomock.Any(), gomock.Any()).Return(domain.User{}, nil)
				return us, cs
			},
			reqBuilder: func(t *testing.T) *http.Request {
				//json格式字符串别忘了key和val都是字符串，要加""
				body := bytes.NewBuffer([]byte(`{"phone":"1314xxx","code":"65422"}`))
				//url路径的前缀/别忘了，表示根路径
				//body接收的是io.reader接口，实现该接口的有bytes.NewBUffer，很好用
				req, err := http.NewRequest(http.MethodPost, "/users/login_sms", body)
				req.Header.Set("Content-Type", "application/json")
				if err != nil {
					t.Fatal(err)
				}
				return req
			},
			wantRes: Result{Msg: "登陆成功"},
		},
		{
			name: "bind失败",
			mock: func(ctrl *gomock.Controller) (service2.UserService, service.CodeService) {
				return nil, nil
			},
			reqBuilder: func(t *testing.T) *http.Request {
				//json格式字符串别忘了key和val都是字符串，要加""
				body := bytes.NewBuffer([]byte(`{"phone":"13`))
				//url路径的前缀/别忘了，表示根路径
				//body接收的是io.reader接口，实现该接口的有bytes.NewBUffer，很好用
				req, err := http.NewRequest(http.MethodPost, "/users/login_sms", body)
				req.Header.Set("Content-Type", "application/json")
				if err != nil {
					t.Fatal(err)
				}
				return req
			},
			wantRes: Result{Msg: "手机号与验证码接收失败"},
		},
		{
			name: "验证码错误",
			mock: func(ctrl *gomock.Controller) (service2.UserService, service.CodeService) {
				us := svcmocks.NewMockUserService(ctrl)
				cs := svcmocks.NewMockCodeService(ctrl)
				cs.EXPECT().Verify(gomock.Any(), "login", gomock.Any(), gomock.Any()).Return(cache.ErrorCodeNotRight)
				return us, cs
			},
			reqBuilder: func(t *testing.T) *http.Request {
				//json格式字符串别忘了key和val都是字符串，要加""
				body := bytes.NewBuffer([]byte(`{"phone":"1314xxx","code":"65422"}`))
				//url路径的前缀/别忘了，表示根路径
				//body接收的是io.reader接口，实现该接口的有bytes.NewBUffer，很好用
				req, err := http.NewRequest(http.MethodPost, "/users/login_sms", body)
				req.Header.Set("Content-Type", "application/json")
				if err != nil {
					t.Fatal(err)
				}
				return req
			},
			wantRes: Result{Msg: "验证码错误!"},
		},
		{
			name: "验证次数过多",
			mock: func(ctrl *gomock.Controller) (service2.UserService, service.CodeService) {
				us := svcmocks.NewMockUserService(ctrl)
				cs := svcmocks.NewMockCodeService(ctrl)
				cs.EXPECT().Verify(gomock.Any(), "login", gomock.Any(), gomock.Any()).Return(cache.ErrorCodeVerifyTooManyTimes)
				return us, cs
			},
			reqBuilder: func(t *testing.T) *http.Request {
				//json格式字符串别忘了key和val都是字符串，要加""
				body := bytes.NewBuffer([]byte(`{"phone":"1314xxx","code":"65422"}`))
				//url路径的前缀/别忘了，表示根路径
				//body接收的是io.reader接口，实现该接口的有bytes.NewBUffer，很好用
				req, err := http.NewRequest(http.MethodPost, "/users/login_sms", body)
				req.Header.Set("Content-Type", "application/json")
				if err != nil {
					t.Fatal(err)
				}
				return req
			},
			wantRes: Result{Msg: "验证次数过多"},
		},
		{
			name: "验证系统错误",
			mock: func(ctrl *gomock.Controller) (service2.UserService, service.CodeService) {
				us := svcmocks.NewMockUserService(ctrl)
				cs := svcmocks.NewMockCodeService(ctrl)
				cs.EXPECT().Verify(gomock.Any(), "login", gomock.Any(), gomock.Any()).Return(errors.New("redis错误"))
				return us, cs
			},
			reqBuilder: func(t *testing.T) *http.Request {
				//json格式字符串别忘了key和val都是字符串，要加""
				body := bytes.NewBuffer([]byte(`{"phone":"1314xxx","code":"65422"}`))
				//url路径的前缀/别忘了，表示根路径
				//body接收的是io.reader接口，实现该接口的有bytes.NewBUffer，很好用
				req, err := http.NewRequest(http.MethodPost, "/users/login_sms", body)
				req.Header.Set("Content-Type", "application/json")
				if err != nil {
					t.Fatal(err)
				}
				return req
			},
			wantRes: Result{Msg: "系统错误"},
		},
		{
			name: "数据库系统错误",
			mock: func(ctrl *gomock.Controller) (service2.UserService, service.CodeService) {
				us := svcmocks.NewMockUserService(ctrl)
				cs := svcmocks.NewMockCodeService(ctrl)
				cs.EXPECT().Verify(gomock.Any(), "login", gomock.Any(), gomock.Any()).Return(nil)
				us.EXPECT().FindOrCreateByPhone(gomock.Any(), gomock.Any()).Return(domain.User{}, errors.New("sql错误"))
				return us, cs
			},
			reqBuilder: func(t *testing.T) *http.Request {
				//json格式字符串别忘了key和val都是字符串，要加""
				body := bytes.NewBuffer([]byte(`{"phone":"1314xxx","code":"65422"}`))
				//url路径的前缀/别忘了，表示根路径
				//body接收的是io.reader接口，实现该接口的有bytes.NewBUffer，很好用
				req, err := http.NewRequest(http.MethodPost, "/users/login_sms", body)
				req.Header.Set("Content-Type", "application/json")
				if err != nil {
					t.Fatal(err)
				}
				return req
			},
			wantRes: Result{Msg: "系统错误"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			us, cs := tc.mock(ctrl)
			uh := NewUserHandler(us, cs, nil)
			server := gin.Default()
			uh.RegisterRouter(server)
			req := tc.reqBuilder(t)
			recoder := httptest.NewRecorder()
			server.ServeHTTP(recoder, req)
			var res Result
			err := json.Unmarshal(recoder.Body.Bytes(), &res)
			assert.NoError(t, err)
			assert.Equal(t, tc.wantRes, res)
		})

	}
}
