package initialize

import (
	"fmt"
	"log"
	"order/global"
	"order/proto"

	_ "github.com/mbobakov/grpc-consul-resolver" //这个必须加上，要不然会有报错
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func InitSrvConn() {
	ConsulInfo := global.ServerConfig.ConsulInfo
	//连接库存服务
	inventoryConn, errs := grpc.NewClient(fmt.Sprintf(
		`consul://%s:%d/%s?wait=14s&tag=%s`, ConsulInfo.Host, ConsulInfo.Port, global.ServerConfig.InventorySrvInfo.Name, global.ServerConfig.InventorySrvInfo.Tags),
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`), // This sets the initial balancing policy.
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if errs != nil {
		log.Fatal(errs)
		zap.S().Fatalf("[InitSrvConn] 连接 【库存服务失败】")
	}
	//拨号连接库存grpc服务
	inventorySrvClient := proto.NewInventoryClient(inventoryConn)
	global.InventoryClient = inventorySrvClient
	//连接商品服务
	goodsConn, err := grpc.NewClient(fmt.Sprintf(
		`consul://%s:%d/%s?wait=14s&tag=%s`, ConsulInfo.Host, ConsulInfo.Port, global.ServerConfig.GoodsSrvInfo.Name, global.ServerConfig.GoodsSrvInfo.Tags),
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`), // This sets the initial balancing policy.
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
		zap.S().Fatalf("[InitSrvConn] 连接 【商品服务失败】")
	}
	//拨号连接用户grpc服务
	goodsSrvClient := proto.NewGoodsClient(goodsConn)
	global.GoodsSrvClient = goodsSrvClient
}
