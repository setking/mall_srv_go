package main

import (
	"context"
	"fmt"
	"goods/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var GoodsClient proto.GoodsClient
var conn *grpc.ClientConn

func start() {
	var err error
	conn, err = grpc.NewClient("192.168.0.102:50035", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	GoodsClient = proto.NewGoodsClient(conn)

}
func TestGoodsList() {
	r, err := GoodsClient.GoodsList(context.Background(), &proto.GoodsFilterRequest{
		Pages:       1,
		PagePerNums: 10,
		KeyWords:    "奥利奥",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(r.Total)
	for _, goods := range r.Data {
		fmt.Println(goods)
	}
}
func TestBatchGetGoods() {
	r, err := GoodsClient.BatchGetGoods(context.Background(), &proto.BatchGoodsIdInfo{
		Id: []int32{421, 422, 423},
	})
	if err != nil {
		panic(err)
	}
	for _, goods := range r.Data {
		fmt.Println(goods)
	}

}
func TestCreateGoods() {
	r, err := GoodsClient.CreateGoods(context.Background(), &proto.CreateGoodsInfo{
		Name:            "奥利奥",
		GoodsSn:         "12312",
		MarketPrice:     12.1,
		ShopPrice:       10.6,
		GoodsBrief:      "测试",
		ShipFree:        true,
		Images:          []string{"asdasd.png", "asdasdasd.jpg"},
		GoodsFrontImage: "asdasdasd.jpg",
		IsNew:           true,
		IsHot:           true,
		OnSale:          false,
		BrandId:         632,
		CategoryId:      135532,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(r.Id)
}
func TestDeleteGoods() {
	_, err := GoodsClient.DeleteGoods(context.Background(), &proto.DeleteGoodsInfo{
		Id: 852,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("success")
}
func TestUpdateGoods() {
	_, err := GoodsClient.UpdateGoods(context.Background(), &proto.CreateGoodsInfo{
		Id:              853,
		Name:            "奥利奥2号",
		GoodsSn:         "12312",
		MarketPrice:     12.1,
		ShopPrice:       10.6,
		GoodsBrief:      "测试",
		ShipFree:        true,
		Images:          []string{"asdasd.png", "asdasdasd.jpg"},
		GoodsFrontImage: "asdasdasd.jpg",
		IsNew:           true,
		IsHot:           true,
		OnSale:          false,
		BrandId:         632,
		CategoryId:      135532,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("success")
}
func TestGetGoodsDetail() {
	rsp, err := GoodsClient.GetGoodsDetail(context.Background(), &proto.GoodInfoRequest{
		Id: 846,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(rsp)

}
func main() {
	start()
	//TestCreateGoods()
	TestUpdateGoods()
	//TestDeleteGoods()
}
