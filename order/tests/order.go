package main

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"order/proto"
)

var OrderClient proto.OrderClient
var conn *grpc.ClientConn

func start() {
	var err error
	conn, err = grpc.NewClient("192.168.0.102:12031", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	OrderClient = proto.NewOrderClient(conn)

}
func TestCartItemList() {
	rsp, err := OrderClient.CartItemList(context.Background(), &proto.UserInfo{
		Id: 1,
	})
	if err != nil {
		panic(err)
	}
	for _, cart := range rsp.Data {
		fmt.Println(cart)
	}
}
func TestCreateCartItem() {
	id, err := OrderClient.CreateCartItem(context.Background(), &proto.CartItemRequest{
		UserId:  1,
		GoodsId: 421,
		Nums:    22,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(id)
}
func TestUpdateCartItem() {
	_, err := OrderClient.UpdateCartItem(context.Background(), &proto.CartItemRequest{
		UserId:  1,
		GoodsId: 421,
		Nums:    1000,
		Checked: true,
	})
	if err != nil {
		panic(err)
	}
}
func TestDeleteCartItem() {
	_, err := OrderClient.DeleteCartItem(context.Background(), &proto.CartItemRequest{
		GoodsId: 423,
		UserId:  1,
	})
	if err != nil {
		panic(err)
	}
}
func TestCreateOrder() {
	info, err := OrderClient.CreateOrder(context.Background(), &proto.OrderRequest{
		UserId:  1,
		Address: "美国加州",
		Name:    "川普",
		Phone:   "18888888888",
		Post:    "MAGA",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(info)
}
func TestOrderList() {
	info, err := OrderClient.OrderList(context.Background(), &proto.OrderFilterRequest{
		UserId:      1,
		Pages:       1,
		PagePerNums: 5,
	})
	if err != nil {
		panic(err)
	}
	for _, data := range info.Data {
		fmt.Println(data)
	}
}
func TestOrderDetail() {
	info, err := OrderClient.OrderDetail(context.Background(), &proto.OrderRequest{
		UserId: 1,
		Id:     15,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(info)
}
func TestUpdateOrderStatus() {
	info, err := OrderClient.UpdateOrder(context.Background(), &proto.OrderStatus{
		OrderSn: "202584198663782700043",
		Status:  "TRADE_SUCCESS",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(info)
}
func main() {
	start()
	TestCreateOrder()
}
