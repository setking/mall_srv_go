package handle

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
	"user/global"
	"user/model"
	"user/proto"
	"user/utils"
)

type UserServer struct {
	proto.UnimplementedUserServer
}

// 获取用户列表
func (u *UserServer) GetUserList(ctx context.Context, req *proto.PageInfo) (*proto.UserListResponse, error) {
	var users []model.User
	result := global.DB.Find(&users)
	if result.Error != nil {
		return nil, result.Error
	}
	rsp := &proto.UserListResponse{}
	rsp.Total = int32(result.RowsAffected)
	global.DB.Scopes(utils.Paginate(int(req.Pn), int(req.PSize))).Find(&users)
	for _, user := range users {
		userInfoRes := utils.ModelToRes(user)
		rsp.Data = append(rsp.Data, &userInfoRes)
	}
	return rsp, nil
}

// 通过手机号查询用户
func (u *UserServer) GetUserByPhone(ctx context.Context, req *proto.PhoneRequest) (*proto.UserInfoResponse, error) {
	var user model.User
	result := global.DB.Where("phone = ?", req.Phone).First(&user)
	if result.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "用户不存在")
	}
	if result.Error != nil {
		return nil, result.Error
	}
	userInfoRes := &proto.UserInfoResponse{
		Id:       int64(user.ID),
		NickName: user.NickName,
		Password: user.Password,
		Phone:    user.Phone,
		Gender:   user.Gender,
		Role:     int32(user.Role),
	}
	return userInfoRes, nil
}

// 通过id查询用户
func (u *UserServer) GetUserByID(ctx context.Context, req *proto.UserIDRequest) (*proto.UserInfoResponse, error) {
	var user model.User
	result := global.DB.First(&user, req.Id)
	if result.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "用户不存在")
	}
	if result.Error != nil {
		return nil, result.Error
	}
	userInfoRes := &proto.UserInfoResponse{
		Id:       int64(user.ID),
		NickName: user.NickName,
		Password: user.Password,
		Phone:    user.Phone,
		Gender:   user.Gender,
		Role:     int32(user.Role),
	}
	return userInfoRes, nil
}

// 创建用户
func (u *UserServer) CreateUser(ctx context.Context, req *proto.CreateUserInfo) (*proto.UserInfoResponse, error) {
	var user model.User
	result := global.DB.Where("phone = ?", req.Phone).First(&user)
	if result.RowsAffected == 1 {
		return nil, status.Errorf(codes.AlreadyExists, "用户已存在")
	}

	user.Phone = req.Phone
	user.NickName = req.NickName
	user.Password = utils.GenMd5(req.Password)
	res := global.DB.Create(&user)
	fmt.Println(user.UpdatedAt)
	if res.Error != nil {
		return nil, status.Errorf(codes.Internal, res.Error.Error())
	}
	rsp := &proto.UserInfoResponse{
		Id: int64(user.ID),
	}
	return rsp, nil
}

// 更新用户信息
func (u *UserServer) UpdateUser(ctx context.Context, req *proto.UpdateUserInfo) (*empty.Empty, error) {
	var user model.User
	result := global.DB.First(&user, req.Id)
	if result.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "用户不存在")
	}

	user.NickName = req.NickName
	user.Gender = req.Gender
	Birthday := time.Unix(int64(req.BirthDay), 0)
	user.Birthday = &Birthday
	res := global.DB.Save(&user)
	if res.Error != nil {
		return nil, status.Errorf(codes.Internal, res.Error.Error())
	}
	return &empty.Empty{}, nil
}

// 检查用户密码
func (u *UserServer) CheckPassword(ctx context.Context, req *proto.PasswordCheckInfo) (*proto.CheckResponse, error) {
	check := utils.VerifyPassword(req.EncryptedPassword, req.Password)
	return &proto.CheckResponse{
		Success: check,
	}, nil
}
