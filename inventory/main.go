package main

import (
	"flag"
	"fmt"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/google/uuid"
	"github.com/hashicorp/consul/api"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"inventory/global"
	"inventory/handle"
	"inventory/initialize"
	"inventory/proto"
	"inventory/utils"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	//初始化日志
	initialize.InitLogger()
	//初始化全局配置
	initialize.InitConfig()
	//初始化连接数据库
	initialize.InitDB()
	//初始化Redis
	initialize.InitRedis()
	IP := flag.String("ip", "0.0.0.0", "ip地址")
	Port := flag.Int("port", 0, "端口号")
	flag.Parse()
	if *Port == 0 {
		*Port, _ = utils.GetFreePort()
	}
	//GlobalPort测试用全局变量端口

	server := grpc.NewServer()
	proto.RegisterInventoryServer(server, &handle.InventoryServer{})
	host := fmt.Sprintf("%s:%d", *IP, *Port)
	lis, err := net.Listen("tcp", host)
	if err != nil {
		panic("filed to listen:" + err.Error())
	}
	//注册服务健康检查
	grpc_health_v1.RegisterHealthServer(server, health.NewServer())
	//服务注册
	cfg := api.DefaultConfig()
	cfg.Address = fmt.Sprintf("%s:%d", global.ServerConfig.ConsulInfo.Host, global.ServerConfig.ConsulInfo.Port)
	client, err := api.NewClient(cfg)
	if err != nil {
		panic(err)
	}
	//配置检查对象
	check := &api.AgentServiceCheck{
		GRPC:                           fmt.Sprintf("%s:%d", global.ServerConfig.Host, *Port),
		Timeout:                        "5s",
		Interval:                       "5s",
		DeregisterCriticalServiceAfter: "15s",
	}

	//生成注册对象
	registration := new(api.AgentServiceRegistration)
	registration.Name = global.ServerConfig.Name
	serviceID := fmt.Sprintf("%s", uuid.New())
	registration.ID = serviceID
	registration.Port = *Port
	registration.Tags = global.ServerConfig.Tags
	registration.Address = global.ServerConfig.Host
	registration.Check = check
	err = client.Agent().ServiceRegister(registration)
	if err != nil {
		panic(err)
	}

	go func() {
		err = server.Serve(lis)
		if err != nil {
			panic("failed to start grpc:" + err.Error())
		}
	}()
	//注册监听rocketmq consumer
	//sig := make(chan os.Signal)
	mqHost := fmt.Sprintf("%s:%d", global.ServerConfig.MqInfo.Host, global.ServerConfig.MqInfo.Port)
	c, _ := rocketmq.NewPushConsumer(
		consumer.WithGroupName(global.ServerConfig.MqInfo.InvGroupName),
		consumer.WithNameServer([]string{mqHost}),
	)
	errs := c.Subscribe("order_reback", consumer.MessageSelector{}, handle.AutoReback)
	if errs != nil {
		fmt.Println(err.Error())
	}
	// Note: start after subscribe
	err = c.Start()
	if err != nil {
		fmt.Println(err.Error())
	}
	//<-sig
	//err = c.Shutdown()
	//if err != nil {
	//	fmt.Printf("Consumer关闭失败: %s", err.Error())
	//}
	//不能让rocketmq主goroutine退出

	//接收终止信号
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	_ = c.Shutdown()
	if err = client.Agent().ServiceDeregister(serviceID); err != nil {
		zap.S().Info("注销失败")
	}
	zap.S().Info("注销成功")
}
