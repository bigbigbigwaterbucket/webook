package web

import (
	"fmt"
	regexp "github.com/dlclark/regexp2"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	jwt "github.com/golang-jwt/jwt/v5"
	"learning_go/webook/internal/domain"
	"learning_go/webook/internal/service"
	"net/http"
	"time"
	//"regexp"
)

// UserHandler 定义跟用户有关的路由
type UserHandler struct {
	svc            *service.UserService
	EmailRegexp    *regexp.Regexp
	PasswordRegexp *regexp.Regexp
	BirthdayRegexp *regexp.Regexp
}

func NewUserHandler(svc *service.UserService) (uh *UserHandler) {
	const (
		emailString    = "^[A-Za-z0-9\\u4e00-\\u9fa5]+@[a-zA-Z0-9_-]+(\\.[a-zA-Z0-9_-]+)+$"
		passwordString = ""
		birthdayString = `^\d{4}-(1[0-2]|0?[1-9])-(3[01]|[12][0-9]|0?[1-9])$`
	)
	return &UserHandler{svc: svc,
		EmailRegexp:    regexp.MustCompile(emailString, 0),
		PasswordRegexp: regexp.MustCompile(passwordString, 0),
		BirthdayRegexp: regexp.MustCompile(birthdayString, 0),
	}
}

func (this *UserHandler) RegisterRouter(engine *gin.Engine) {
	//注册路由组，集中处理相同前缀的路由
	ug := engine.Group("/users")
	ug.POST("/signup", this.SignUp)
	ug.POST("/login", this.LoginJWT)
	ug.POST("/edit", this.Edit)
	ug.GET("/profile", this.ProfileJWT)
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
		if err == service.ErrUserDuplicateEmail {
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

	//60秒后过期，jwt的valid将变为false
	claims := UserClaims{RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute))},
		Uid:       user.Id,
		UserAgent: ctx.Request.UserAgent()}
	//生成带自定义信息的token
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	tokenStr, err := token.SignedString([]byte("1234567890abcdef1234567890abcdef"))
	if err != nil {
		ctx.String(http.StatusInternalServerError, "系统错误")
	}
	ctx.Header("x-jwt-token", tokenStr)
	fmt.Printf(tokenStr)
	fmt.Printf("%v", user)
	ctx.String(http.StatusOK, "登录成功")
	return
}

type UserClaims struct {
	jwt.RegisteredClaims
	Uid       int64
	UserAgent string
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
