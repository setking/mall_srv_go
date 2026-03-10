### mall_srv
一个基于Go + Gorm、Nacos、Consul 、Redis、RocketMQ、Grpc, jaeger、Elasticsearch，采用主流的互联网技术架构、支持集群部署、服务注册和发现以及拥有完整的订单流程等，代码完全开源，没有任何二次封装。



### 特性
- 基于Gorm框架，提供了统一方便管理的数据库访问
- 使用grpc作为微服务通信基础
- 使用zap作为日志库
- 使用consul作为注册中心和服务发现
- 使用nacos作为配置中心
- 使用redis保证库存扣减流程的原子性
- 使用RocketMQ作为消息中间件，事务消息结合Redis分布式锁保证订单创建与消息发送的原子性，通过异步消息驱动下游状态更新，配合消费端幂等和定时对账，实现了订单流程的最终一致性。
- 接入jaeger+OpenTelemetry链路追供功能
- 使用Elasticsearch作为商品搜索服务
- 接入jenkins实现自动化部署

### 基本功能
- 用户服务
- 库存服务
- 商品服务
- 订单服务
- 购物车服务

### 目录结构

```azure
mall_srv
goods -- 商品服务
    ├─handle  -- 服务api接口配置
    ├─config  -- nacos配置
    ├─global  -- 全局配置
    ├─initialize  -- 初始化配置
    ├─middlewares  -- 服务插件配置
    ├─model  -- 数据库表数据配置
    ├─proto  -- protobuf文件存储地址
    ├─tmp  -- nacos日志存储
    ├─logs  -- zap日志存储
    └─utils  -- 工具函数
    └─main.go  -- 服务入口


order -- 订单服务
    ├─handle  -- 服务api接口配置
    ├─config  -- nacos配置
    ├─global  -- 全局配置
    ├─initialize  -- 初始化配置
    ├─middlewares  -- 服务插件配置
    ├─model  -- 数据库表数据配置
    ├─proto  -- protobuf文件存储地址
    ├─tmp  -- nacos日志存储
    ├─logs  -- zap日志存储
    └─utils  -- 工具函数
    └─main.go  -- 服务入口
    
inventory -- 库存服务
    ├─handle  -- 服务api接口配置
    ├─config  -- nacos配置
    ├─global  -- 全局配置
    ├─initialize  -- 初始化配置
    ├─middlewares  -- 服务插件配置
    ├─model  -- 数据库表数据配置
    ├─proto  -- protobuf文件存储地址
    ├─tmp  -- nacos日志存储
    ├─logs  -- zap日志存储
    └─utils  -- 工具函数
    └─main.go  -- 服务入口
  
user -- 用户服务
    ├─handle  -- 服务api接口配置
    ├─config  -- nacos配置
    ├─global  -- 全局配置
    ├─initialize  -- 初始化配置
    ├─middlewares  -- 服务插件配置
    ├─model  -- 数据库表数据配置
    ├─proto  -- protobuf文件存储地址
    ├─tmp  -- nacos日志存储
    ├─logs  -- zap日志存储
    └─utils  -- 工具函数
    └─main.go  -- 服务入口
    
userop_web -- 正在开发中
```
