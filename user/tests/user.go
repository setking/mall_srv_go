package main

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"user/proto"
)

var UserClient proto.UserClient
var conn *grpc.ClientConn

func start() {
	var err error
	conn, err = grpc.NewClient("127.0.0.1:8089", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	UserClient = proto.NewUserClient(conn)

}
func TestGetUserList() {
	r, err := UserClient.GetUserList(context.Background(), &proto.PageInfo{
		Pn:    1,
		PSize: 5,
	})
	if err != nil {
		panic(err)
	}
	for _, user := range r.Data {
		fmt.Println(user.Phone, user.NickName, user.Password)
		rsp, err := UserClient.CheckPassword(context.Background(), &proto.PasswordCheckInfo{
			Password:          "admin121",
			EncryptedPassword: user.Password,
		})
		if err != nil {
			panic(err)
		}
		fmt.Println(rsp.Success)
	}
}
func TestCreateUser() {
	for i := 0; i < 10; i++ {
		r, err := UserClient.CreateUser(context.Background(), &proto.CreateUserInfo{
			NickName: fmt.Sprintf("李雷%d", i),
			Phone:    fmt.Sprintf("1825608876%d", i),
			Password: fmt.Sprintf("admin12%d", i),
		})
		if err != nil {
			panic(err)
		}
		fmt.Println(r.GetId())
	}

}
func TestGetUserByPhone() {
	r, err := UserClient.GetUserByPhone(context.Background(), &proto.PhoneRequest{
		Phone: "18256088763",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(r)
}
func TestGetUserByID() {
	r, err := UserClient.GetUserByID(context.Background(), &proto.UserIDRequest{
		Id: 6,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(r)
}
func TestUpdateUser() {
	r, err := UserClient.UpdateUser(context.Background(), &proto.UpdateUserInfo{
		Id:       2,
		NickName: "吴用",
		Gender:   "female",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(r)
}
func TestCheckPassword() {
	r, err := UserClient.CheckPassword(context.Background(), &proto.PasswordCheckInfo{
		Password:          "admin123",
		EncryptedPassword: "$pbkdf2-sha256$29000$UArhHOMcYywFgBBCqNW6Nw$hkBqJxE8cAElSSLPEzqPf7wAfxYycanI/czTiLWlYls",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(r)
}
func main() {
	start()
	//defer conn.Close()
	//TestGetUserList()
	//TestCreateUser()
	//TestGetUserByPhone()
	//TestGetUserByID()
	//TestUpdateUser()
	TestCheckPassword()
}
