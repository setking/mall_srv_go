package main

import (
	"flag"
	"fmt"
	"inventory/global"
	"inventory/handle"
	"inventory/initialize"
	"inventory/proto"
	"inventory/utils"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/google/uuid"
	"github.com/hashicorp/consul/api"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
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

	server := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     15 * time.Minute, // 连接空闲超过15分钟才关闭
			MaxConnectionAge:      30 * time.Minute, // 连接最长存活30分钟后优雅关闭
			MaxConnectionAgeGrace: 5 * time.Second,  // 优雅关闭等待时间
			Time:                  10 * time.Second, // 每10秒发一次心跳
			Timeout:               3 * time.Second,  // 心跳超时3秒
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second, // 客户端最小心跳间隔
			PermitWithoutStream: true,            // 没有活跃请求时也允许心跳
		}))
	proto.RegisterInventoryServer(server, &handle.InventoryServer{})
	host := fmt.Sprintf("%s:%d", *IP, *Port)
	lis, err := net.Listen("tcp", host)
	if err != nil {
		panic("filed to listen:" + err.Error())
	}
	//注册服务健康检查
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(server, healthServer)
	// 设置为健康状态，Consul 才能检查通过
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// 优雅退出时设置为不健康，让 Consul 提前摘流
	defer healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
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
		Interval:                       "10s", // 检查间隔拉长，减少对服务的干扰
		DeregisterCriticalServiceAfter: "5m",  // 给足压测抖动的容忍时间
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
	err = c.Subscribe("order_reback", consumer.MessageSelector{}, handle.AutoReback)
	if err != nil {
		zap.S().Fatalf("Consumer Subscribe 失败: %v", err)
	}
	// Note: start after subscribe
	err = c.Start()
	if err != nil {
		zap.S().Fatalf("Consumer Start 失败: %v", err)
	}
	//<-sig
	//err = c.Shutdown()
	//if err != nil {
	//	fmt.Printf("Consumer关闭失败: %s", err.Error())
	//}
	//不能让rocketmq主goroutine退出

	//接收终止信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	zap.S().Info("收到退出信号，开始优雅退出...")
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	if err = client.Agent().ServiceDeregister(serviceID); err != nil {
		zap.S().Info("注销失败")
	}
	zap.S().Info("注销成功")
	// RocketMQ 优雅退出
	// 停止 Producer，等待正在发送中的消息全部确认
	if err = c.Shutdown(); err != nil {
		zap.S().Errorf("RocketMQ Producer 关闭失败: %v", err)
	} else {
		zap.S().Info("RocketMQ Producer 关闭成功")
	}
	server.GracefulStop()
	zap.S().Info("gRPC 服务优雅退出完成")
}
