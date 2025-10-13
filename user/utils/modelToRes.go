package utils

import (
	"user/model"
	"user/proto"
)

func ModelToRes(user model.User) proto.UserInfoResponse {
	userInfo := proto.UserInfoResponse{
		Id:       int64(user.ID),
		NickName: user.NickName,
		Password: user.Password,
		Phone:    user.Phone,
		Gender:   user.Gender,
		Role:     int32(user.Role),
	}
	if user.Birthday != nil {
		userInfo.BirthDay = uint64(user.Birthday.Unix())
	}
	return userInfo
}
