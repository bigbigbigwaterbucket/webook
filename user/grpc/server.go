package grpc

import (
	"context"
	"learning_go/webook/api/proto/gen/user/userv1"
	"learning_go/webook/user/domain"
	"learning_go/webook/user/service"
	"time"
)

type UserServiceServer struct {
	userv1.UnimplementedUserServiceServer
	svc service.UserService
}

func NewUserServiceServer(svc service.UserService) *UserServiceServer {
	return &UserServiceServer{svc: svc}
}

func (u *UserServiceServer) Login(ctx context.Context, req *userv1.LoginReq) (*userv1.LoginResp, error) {
	user, err := u.svc.Login(ctx, u.userToDomain(req.GetUser()))
	return &userv1.LoginResp{User: u.userToDao(user)}, err
}

func (u *UserServiceServer) SignUp(ctx context.Context, req *userv1.SignUpReq) (*userv1.SignUpResp, error) {
	err := u.svc.SignUp(ctx, u.userToDomain(req.GetUser()))
	return &userv1.SignUpResp{}, err
}

func (u *UserServiceServer) Edit(ctx context.Context, req *userv1.EditReq) (*userv1.EditResp, error) {
	err := u.svc.Edit(ctx, u.userToDomain(req.GetUser()))
	return &userv1.EditResp{}, err
}

func (u *UserServiceServer) Profile(ctx context.Context, req *userv1.ProfileReq) (*userv1.ProfileResp, error) {
	user, err := u.svc.Profile(ctx, req.GetUserId())
	return &userv1.ProfileResp{User: u.userToDao(user)}, err
}

func (u *UserServiceServer) FindOrCreateByPhone(ctx context.Context, req *userv1.FindOrCreateByPhoneReq) (*userv1.FindOrCreateByPhoneResp, error) {
	user, err := u.svc.FindOrCreateByPhone(ctx, req.GetPhone())
	return &userv1.FindOrCreateByPhoneResp{User: u.userToDao(user)}, err
}

func (u *UserServiceServer) FindOrCreateByWechat(ctx context.Context, req *userv1.FindOrCreateByWechatReq) (*userv1.FindOrCreateByWechatResp, error) {
	user, err := u.svc.FindOrCreateByWechat(ctx, u.infoToDomain(req.GetInfo()))
	return &userv1.FindOrCreateByWechatResp{User: u.userToDao(user)}, err
}

func (u *UserServiceServer) userToDomain(user *userv1.User) domain.User {
	//rpc传输的ctime是unixMilli类型，这要求客户端调用时把time类型转为unixMilli
	return domain.User{Id: user.Id, Nickname: user.Nickname, Email: user.Email, Phone: user.Phone, Password: user.Password,
		Name: user.Name, Birthday: user.Birthday, Introduce: user.Introduce, Ctime: time.UnixMilli(user.Ctime), WechatInfo: u.infoToDomain(user.WechatInfo)}
}

func (u *UserServiceServer) infoToDomain(info *userv1.WechatInfo) domain.WechatInfo {
	if info == nil { //小心，有的请求是不带info的，这里是nil
		return domain.WechatInfo{}
	}
	return domain.WechatInfo{OpenId: info.OpenId, UnionId: info.UnionId}
}

func (u *UserServiceServer) userToDao(user domain.User) *userv1.User {
	return &userv1.User{Id: user.Id, Nickname: user.Nickname, Email: user.Email, Phone: user.Phone, Password: user.Password,
		Name: user.Name, Birthday: user.Birthday, Introduce: user.Introduce, Ctime: user.Ctime.UnixMilli(), WechatInfo: u.infoToDao(user.WechatInfo)}
}

func (u *UserServiceServer) infoToDao(info domain.WechatInfo) *userv1.WechatInfo {
	return &userv1.WechatInfo{OpenId: info.OpenId, UnionId: info.UnionId}
}
