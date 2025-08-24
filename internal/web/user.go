package web

import (
	"fmt"
	regexp "github.com/dlclark/regexp2"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"learning_go/webook/internal/domain"
	"learning_go/webook/internal/repository/cache"
	"learning_go/webook/internal/service"
	"learning_go/webook/internal/web/ijwt"
	"net/http"
	//"regexp"
)

const biz = "login"

// 通过接口，让编译器去检查UserHandler，确保其实现了注册路由的方法
var _ handler = (*UserHandler)(nil)

// UserHandler 定义跟用户有关的路由
type UserHandler struct {
	svc            service.UserService
	codeSvc        service.CodeService
	EmailRegexp    *regexp.Regexp
	PasswordRegexp *regexp.Regexp
	BirthdayRegexp *regexp.Regexp
	ijwt.JwtHandler
}

func NewUserHandler(svc service.UserService, codeSvc service.CodeService, jwt ijwt.JwtHandler) (uh *UserHandler) {
	const (
		emailString    = "^[A-Za-z0-9\\u4e00-\\u9fa5]+@[a-zA-Z0-9_-]+(\\.[a-zA-Z0-9_-]+)+$"
		passwordString = ""
		birthdayString = `^\d{4}-(1[0-2]|0?[1-9])-(3[01]|[12][0-9]|0?[1-9])$`
	)
	return &UserHandler{svc: svc, codeSvc: codeSvc,
		EmailRegexp:    regexp.MustCompile(emailString, 0),
		PasswordRegexp: regexp.MustCompile(passwordString, 0),
		BirthdayRegexp: regexp.MustCompile(birthdayString, 0),
		JwtHandler:     jwt,
	}
}

func (this *UserHandler) RegisterRouter(engine *gin.Engine) {
	//注册路由组，集中处理相同前缀的路由
	//每在这里注册路由的时候，都要考虑一下跨域问题，如果是登录业务，那么不需要被jwt校验
	ug := engine.Group("/users")
	ug.POST("/signup", this.SignUp)
	ug.POST("/login", this.LoginJWT)
	ug.POST("/edit", this.EditJWT)
	ug.GET("/profile", this.ProfileJWT)
	ug.POST("/logout", this.LogOutJWT)
	ug.POST("/login_sms/code/send", this.SendSmsCode)
	ug.POST("/login_sms", this.LoginSmsCode)
	ug.POST("/refresh_token", this.RefreshToken)
}

func (this *UserHandler) LogOutJWT(ctx *gin.Context) {
	err := this.ClearToken(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, Result{Code: 5, Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, Result{
		Msg: "成功退出登录",
	})
}

func (this *UserHandler) RefreshToken(ctx *gin.Context) {
	tokenStr := this.GetTokenFromAuth(ctx)
	claims := &ijwt.RefreshClaims{}
	//函数规范：传指针就会改指针指向的值，是写；传值就是只读
	tokenReal, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
		return ijwt.RtKey, nil
	})
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, Result{Code: 4, Msg: "请登录"})
		return
	}
	err = this.CheckSsid(ctx, claims.Ssid)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, Result{Code: 4, Msg: "请登录"})
		return
	}
	if tokenReal == nil || !tokenReal.Valid || claims.Uid == 0 {
		ctx.JSON(http.StatusUnauthorized, Result{Code: 4, Msg: "请登录"})
		return
	}
	err = this.SetJwtToken(ctx, claims.Uid, claims.Ssid)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, Result{Code: 4, Msg: "请登录"})
		return
	}
}

func (this *UserHandler) LoginSmsCode(ctx *gin.Context) {
	type Req struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	var req Req
	err := ctx.Bind(&req)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Msg: "手机号与验证码接收失败"})
		return
	}
	err = this.codeSvc.Verify(ctx, biz, req.Phone, req.Code)
	if err != nil {
		if err == cache.ErrorCodeNotRight {
			ctx.JSON(http.StatusOK, Result{Msg: "验证码错误!"})
			return
		}
		if err == cache.ErrorCodeVerifyTooManyTimes {
			ctx.JSON(http.StatusOK, Result{Msg: "验证次数过多"})
			return
		}
		ctx.JSON(http.StatusOK, Result{Msg: "系统错误"})
		return
	}
	user, err := this.svc.FindOrCreateByPhone(ctx, req.Phone)
	if err != nil {
		//ctx.json返回的是结构体序列化后的json串
		ctx.JSON(http.StatusOK, Result{Msg: "系统错误"})
		return
	}
	err = this.SetLoginToken(ctx, user.Id)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{Msg: "系统错误"})
		return
	}
	ctx.JSON(http.StatusOK, Result{Msg: "登陆成功"})

}

func (this *UserHandler) SendSmsCode(ctx *gin.Context) {
	type SmsReq struct {
		Phone string `json:"phone"`
	}
	//var 给一切变量本身分配内存，但如果变量是引用类型（指针、slice、map、chan），它只会初始化为零值（nil），不会自动为其引用的底层数据结构分配内存，需要 new 或 make 来分配。
	var req SmsReq // 值类型变量，已经在栈上分配好内存
	err := ctx.Bind(&req)
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	err = this.codeSvc.Send(ctx, biz, req.Phone)
	if err != nil {
		if err == cache.ErrorCodeSendTooMany {
			zap.L().Warn("验证码发送太频繁", zap.String("biz", biz), zap.String("phone", req.Phone))
			//可以在告警系统里配置，如果上述warn一分钟出现100次，就告警
			ctx.JSON(http.StatusOK, Result{Msg: "验证码发送太频繁"})
			return
		}
		ctx.JSON(http.StatusOK, Result{Msg: "验证码发送失败"})
		return
	}
	ctx.JSON(http.StatusOK, Result{Msg: "发送成功"})
}

func (this *UserHandler) SignUp(ctx *gin.Context) {
	//内部结构体，防止其他函数使用
	type SignUpReq struct {
		Email           string "json:\"email\"" //结构体标签，用于结构体序列化成json格式数据，根据json的键赋值，这里如果不写，键就得是Email
		Password        string `json:"password"`
		ConfirmPassword string `json:"confirmPassword"`
	}
	var req SignUpReq
	//Bind方法会根据 请求体的类型Content-type通常是application/json 尝试把返回的参数序列化后，传给结构体，成功就不会报错
	if err := ctx.Bind(&req); err != nil {
		//这里返回码会强制变成400，可能与gin框架有关
		ctx.String(http.StatusOK, "json格式错误")
		return
	}
	ok, err := this.EmailRegexp.MatchString(req.Email)
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	if !ok {
		ctx.String(http.StatusOK, "邮箱格式错误")
		return
	}

	if req.Password != req.ConfirmPassword {
		ctx.String(http.StatusOK, "两次输入密码不一致")
		return
	}

	ok, err = this.PasswordRegexp.MatchString(req.Password)
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	if !ok {
		ctx.String(http.StatusOK, "密码格式错误")
		return
	}

	err = this.svc.SignUp(ctx, domain.User{Email: req.Email, Password: req.Password})

	if err != nil {
		//这个eer其实包含了提示字符串。。。
		if err == service.ErrUserDuplicate {
			ctx.String(200, "邮箱冲突")
		} else {
			ctx.String(http.StatusOK, "系统错误")
		}
		return
	}
	ctx.String(http.StatusOK, "注册成功")
	fmt.Printf("%v", req)
}

func (this *UserHandler) LoginJWT(ctx *gin.Context) {
	type LoginReq struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	var req LoginReq
	if err := ctx.Bind(&req); err != nil {
		return
	}
	user, err := this.svc.Login(ctx, domain.User{Email: req.Email, Password: req.Password})
	if err != nil {
		if err == service.ErrInvalidUserOrPassword {
			ctx.String(http.StatusOK, "用户名或密码错误")
			return
		} else {
			ctx.String(http.StatusOK, "系统错误")
		}
	}

	err = this.SetLoginToken(ctx, user.Id)
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	ctx.String(http.StatusOK, "登录成功")
	return
}

func (this *UserHandler) Login(ctx *gin.Context) {
	type LoginReq struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	var req LoginReq
	if err := ctx.Bind(&req); err != nil {
		return
	}
	user, err := this.svc.Login(ctx, domain.User{Email: req.Email, Password: req.Password})
	if err != nil {
		if err == service.ErrInvalidUserOrPassword {
			ctx.String(http.StatusOK, "用户名或密码错误")
			return
		} else {
			ctx.String(http.StatusOK, "系统错误")
		}
	}

	//设置session
	sess := sessions.Default(ctx)
	//设置放在session里的值
	sess.Set("userId", user.Id)
	sess.Options(sessions.Options{
		MaxAge: 30 * 60, //max age除了控制cookie中的ssid字段，还控制redis里key value值的过期时间
		// Secure: true, 只能通过https协议访问才能设置session_id和session登录信息
	}) //options限制的是cookie中的ssid字段的存在时间等等
	err = sess.Save()
	if err != nil {
		return
	}
	ctx.String(http.StatusOK, "登录成功")
	return
}

func (this *UserHandler) LogOut(ctx *gin.Context) {
	sess := sessions.Default(ctx)
	sess.Options(sessions.Options{
		MaxAge: -1, //max age除了控制cookie中的ssid字段
		// Secure: true, 只能通过https协议访问才能设置session_id和session登录信息
	}) //options限制的是cookie中的ssid字段的存在时间等等
	err := sess.Save()
	if err != nil {
		return
	}
}

func (this *UserHandler) Profile(ctx *gin.Context) {
	type UserResp struct {
		Name      string `json:"name"` //添加标签作为json的键，如果不加，默认是结构体的字段名
		Birthday  string `json:"birthday"`
		Introduce string `json:"introduce"`
	}
	sess := sessions.Default(ctx)
	userId, ok := sess.Get("userId").(int64)
	if !ok {
		ctx.String(http.StatusOK, "没有登录信息!")
		return
	}
	user, err := this.svc.Profile(ctx, userId)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "获取用户信息失败")
		return
	}
	resp := UserResp{
		Name:      user.Name,
		Birthday:  user.Birthday,
		Introduce: user.Introduce,
	}

	ctx.JSON(http.StatusOK, resp)
	return
}

func (this *UserHandler) ProfileJWT(ctx *gin.Context) {
	type UserResp struct {
		Name      string `json:"name"` //添加标签作为json的键，如果不加，默认是结构体的字段名
		Birthday  string `json:"birthday"`
		Introduce string `json:"introduce"`
	}
	//重新解析token
	//token := ctx.GetHeader("Authorization")
	//segs := strings.SplitN(token, " ", 2)
	//if len(segs) != 2 {
	//	ctx.AbortWithStatus(http.StatusUnauthorized)
	//	return
	//}
	//tokenStr := segs[1]
	//claims := &UserClaims{}
	//tokenReal, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
	//	return []byte("1234567890abcdef1234567890abcdef"), nil
	//})
	//if err != nil || tokenReal.Valid == false {
	//	ctx.String(http.StatusOK, "没有登录信息!")
	//	return
	//}
	userId, ok := ctx.Get("userId")
	//不需要下面这个判断，因为如果拿不到下面也会捕获到错误
	//if !ok {
	//	ctx.String(http.StatusInternalServerError, "获取用户信息失败")
	//	return
	//}
	userIdReal, ok := userId.(int64)
	if !ok {
		ctx.String(http.StatusInternalServerError, "获取用户信息失败")
		return
	}
	user, err := this.svc.Profile(ctx, userIdReal)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "获取用户信息失败")
		return
	}
	resp := UserResp{
		Name:      user.Name,
		Birthday:  user.Birthday,
		Introduce: user.Introduce,
	}

	ctx.JSON(http.StatusOK, resp)
	return
}

func (this *UserHandler) Edit(ctx *gin.Context) {
	type EditReq struct {
		Name      string `json:"name"` //必须是大写，保证json格式数据转结构体时能够赋值，但是tag的json键不是大小写敏感的
		Birthday  string `json:"birthday"`
		Introduce string `json:"introduce"`
	}
	//go语言的处理错误的方式已经深刻融入至编程中，正常每写一个函数都要try catch，在go里。由于没有try catch，返回的就是error或nil类型，强制你去处理
	var req EditReq
	err := ctx.Bind(&req)
	if err != nil {
		return
	}
	//处理输入的信息是否合法
	if len(req.Name) > 20 || len(req.Name) == 0 {
		ctx.String(http.StatusOK, "用户名过长或为空")
		return
	}
	ok, err := this.BirthdayRegexp.MatchString(req.Birthday)
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	if !ok {
		ctx.String(http.StatusOK, "生日日期不符合格式")
		return
	}
	if len(req.Introduce) > 50 {
		ctx.String(http.StatusOK, "个人简介过长")
		return
	}

	sess := sessions.Default(ctx)
	//go的interface{}/any类型在进行类型转换前必须要先断言，其他基本类型互相转化不用断言
	//userId := int64(sess.Get("userId"))
	userId, ok := sess.Get("userId").(int64)
	if !ok {
		ctx.String(http.StatusOK, "没有登录信息!")
		return
	}
	//这里从session中找到userId,后续存储数据时就可以找到是哪个用户
	err = this.svc.Edit(ctx, domain.User{Id: userId, Name: req.Name, Birthday: req.Birthday, Introduce: req.Introduce})
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	ctx.String(http.StatusOK, "用户信息更改成功！")
	return
}

func (this *UserHandler) EditJWT(ctx *gin.Context) {
	type EditReq struct {
		Name      string `json:"nickname"` //必须是大写，保证json格式数据转结构体时能够赋值，但是tag的json键不是大小写敏感的
		Birthday  string `json:"birthday"`
		Introduce string `json:"aboutMe"`
	}
	//go语言的处理错误的方式已经深刻融入至编程中，正常每写一个函数都要try catch，在go里。由于没有try catch，返回的就是error或nil类型，强制你去处理
	var req EditReq
	err := ctx.Bind(&req)
	if err != nil {
		return
	}
	//处理输入的信息是否合法
	if len(req.Name) > 20 || len(req.Name) == 0 {
		ctx.String(http.StatusOK, "用户名过长或为空")
		return
	}
	ok, err := this.BirthdayRegexp.MatchString(req.Birthday)
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	if !ok {
		ctx.String(http.StatusOK, "生日日期不符合格式")
		return
	}
	if len(req.Introduce) > 50 {
		ctx.String(http.StatusOK, "个人简介过长")
		return
	}

	userId, ok := ctx.Get("userId")
	if !ok {
		ctx.String(http.StatusUnauthorized, "尚未登录")
	}
	userReal, ok := userId.(int64)
	if !ok {
		ctx.String(http.StatusUnauthorized, "尚未登录")
	}
	//这里从session中找到userId,后续存储数据时就可以找到是哪个用户
	err = this.svc.Edit(ctx, domain.User{Id: userReal, Name: req.Name, Birthday: req.Birthday, Introduce: req.Introduce})
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	ctx.String(http.StatusOK, "用户信息更改成功！")
	return
}
