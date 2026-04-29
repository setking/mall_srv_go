# Mall Srv Go

<p align="center">
  <a href="#"><img src="https://img.shields.io/badge/Go-1.18+-00ADD8?style=flat-square&logo=go" alt="Go Version"></a>
  <a href="#"><img src="https://img.shields.io/badge/gRPC-1.50+-00ADD8?style=flat-square" alt="gRPC"></a>
  <a href="#"><img src="https://img.shields.io/badge/Protocol%20Buffers-3.21+-00ADD8?style=flat-square" alt="Protobuf"></a>
  <a href="#"><img src="https://img.shields.io/badge/Nacos-2.0+-00ADD8?style=flat-square" alt="Nacos"></a>
  <a href="#"><img src="https://img.shields.io/badge/Consul-1.12+-00ADD8?style=flat-square" alt="Consul"></a>
  <a href="#"><img src="https://img.shields.io/badge/Redis-6.0+-00ADD8?style=flat-square" alt="Redis"></a>
  <a href="#"><img src="https://img.shields.io/badge/RocketMQ-4.9+-00ADD8?style=flat-square" alt="RocketMQ"></a>
  <a href="#"><img src="https://img.shields.io/badge/ES-7.x-00ADD8?style=flat-square" alt="Elasticsearch"></a>
</p>

<p align="center">
  基于 Go + gRPC 的微服务电商后端系统 | 分布式 | 高可用 | 可扩展
</p>

---

## ✨ 技术亮点

本项目在技术实现上具有以下核心亮点：

### 🔐 分布式事务与最终一致性

| 技术方案              | 实现细节                                 |
| --------------------- | ---------------------------------------- |
| **RocketMQ 事务消息** | 半消息机制确保本地事务与消息发送的原子性 |
| **Redis 分布式锁**    | 库存扣减的互斥控制，防止超卖             |
| **消息消费幂等**      | 消费者端实现幂等校验，避免重复处理       |
| **定时对账机制**      | 定时任务核对订单状态，确保数据一致性     |

> **核心流程**：订单创建 → 锁定库存 → 发送事务消息 → 下游消费更新状态 → 支付回调 → 库存扣减

### 📡 高性能服务通信

- **gRPC + Protobuf**：二进制序列化，跨语言、跨平台，相比 JSON 性能提升 5-10 倍
- **HTTP/gRPC 双协议支持**：同时暴露 RESTful API 和 gRPC 接口
- **连接池复用**：gRPC 客户端连接池，避免频繁创建连接开销

### 🛡️ 高可用与容错

- **服务注册与发现**：Consul 集群实现服务健康检查与自动摘除
- **配置热更新**：Nacos 配置中心支持配置动态刷新，无需重启服务
- **链路追踪**：Jaeger + OpenTelemetry 全链路分布式追踪，快速定位问题

### 🔍 商品搜索优化

- **Elasticsearch 全文检索**：支持分词、过滤、聚合查询
- **数据同步**：商品数据变更实时同步至 ES
- **分页优化**：深度分页游标机制，支持海量数据高效翻页

### 📊 可观测性

- **结构化日志**：Zap 日志库，支持 JSON 格式输出与日志分级
- **多维度监控**：请求耗时、QPS、错误率等指标采集
- **全链路追踪**：请求全流程追踪，定位性能瓶颈

---

## 🏗️ 系统架构

```
┌─────────────────────────────────────────────────────────────────┐
│                         API Gateway                              │
└─────────────────────────────────────────────────────────────────┘
                                │
        ┌───────────┬───────────┼───────────┬───────────┐
        ▼           ▼           ▼           ▼           ▼
   ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐
   │  User   │ │ Goods   │ │Inventory│ │  Order  │ │ UserOp  │
   │ Service │ │ Service │ │ Service │ │ Service │ │ Service │
   └─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────┘
        │           │           │           │
        └───────────┴─────┬─────┴───────────┘
                         ▼
        ┌────────────────────────────────────────┐
        │           Data Layer                   │
        │  MySQL │ Redis │ Elasticsearch │ MQ    │
        └────────────────────────────────────────┘
```

---

## 📖 项目简介

`mall_srv_go` 是一个基于 **Go 语言**开发的微服务电商后端系统，采用主流的互联网技术架构，支持集群部署、服务注册与发现，提供完整的订单业务流程。

### 核心特性

| 特性           | 说明                              |
| -------------- | --------------------------------- |
| **微服务架构** | 基于 gRPC 实现服务间通信          |
| **服务治理**   | Consul 注册中心 + Nacos 配置中心  |
| **高并发处理** | Redis 分布式锁保证库存扣减原子性  |
| **消息队列**   | RocketMQ 事务消息实现最终一致性   |
| **链路追踪**   | Jaeger + OpenTelemetry 全链路监控 |
| **日志管理**   | Zap 高性能日志库                  |
| **搜索服务**   | Elasticsearch 商品全文搜索        |
| **持续集成**   | Jenkins 自动化部署支持            |

---

## 📦 服务模块

| 服务        | 说明                                |
| ----------- | ----------------------------------- |
| `user`      | 用户服务：注册、登录、用户信息管理  |
| `goods`     | 商品服务：商品 CRUD、分类管理、搜索 |
| `inventory` | 库存服务：库存扣减、锁定、归还      |
| `order`     | 订单服务：订单创建、支付、状态流转  |
| `userop`    | 用户运营服务（开发中）              |

---

## 🛠️ 技术栈

### 核心框架

- **Go 1.18+** - 高性能并发语言
- **gRPC 1.50+** - 高效 RPC 通信框架
- **Protocol Buffers** - 序列化与接口定义
- **GORM** - Go ORM 框架

### 中间件

| 中间件            | 版本  | 用途                 |
| ----------------- | ----- | -------------------- |
| **Nacos**         | 2.0+  | 配置中心，支持热更新 |
| **Consul**        | 1.12+ | 服务注册与发现       |
| **Redis**         | 6.0+  | 缓存 + 分布式锁      |
| **RocketMQ**      | 4.9+  | 事务消息队列         |
| **Elasticsearch** | 7.x   | 全文搜索             |
| **Jaeger**        | 1.x   | 链路追踪             |

### 基础设施

- **MySQL 5.7+** - 持久化存储
- **Zap** - 结构化日志
- **Jenkins** - CI/CD 持续集成

---

## 📂 目录结构

```
mall_srv_go/
├── goods/              # 商品服务
│   ├── handle/         # HTTP/gRPC 接口处理
│   ├── config/         # Nacos 配置解析
│   ├── global/         # 全局变量
│   ├── initialize/     # 初始化逻辑
│   ├── model/          # 数据模型
│   ├── proto/          # Protobuf 定义
│   ├── utils/          # 工具函数
│   └── main.go         # 服务入口
│
├── order/              # 订单服务
│   ├── handle/
│   ├── config/
│   ├── global/
│   ├── initialize/
│   ├── model/
│   ├── proto/
│   ├── utils/
│   └── main.go
│
├── inventory/          # 库存服务
│   ├── handle/
│   ├── config/
│   ├── global/
│   ├── initialize/
│   ├── model/
│   ├── proto/
│   ├── utils/
│   └── main.go
│
├── user/               # 用户服务
│   ├── handle/
│   ├── config/
│   ├── global/
│   ├── initialize/
│   ├── model/
│   ├── proto/
│   ├── utils/
│   └── main.go
│
└── userop/             # 用户运营服务（开发中）
```

### 目录说明

| 目录          | 说明                                |
| ------------- | ----------------------------------- |
| `handle/`     | API 接口处理层                      |
| `config/`     | Nacos 配置解析                      |
| `global/`     | 全局配置与变量                      |
| `initialize/` | 初始化配置（数据库、Redis、日志等） |
| `model/`      | 数据库表结构定义                    |
| `proto/`      | Protobuf 协议文件                   |
| `utils/`      | 工具函数                            |
| `logs/`       | 日志存储目录                        |
| `tmp/`        | 临时文件（Nacos 日志等）            |

---

## 🚀 快速开始

### 环境要求

- Go 1.18+
- MySQL 5.7+
- Redis 6.0+
- Consul 1.12+
- Nacos 2.0+
- RocketMQ 4.9+
- Elasticsearch 7.x
- Jaeger 1.x

### 配置说明

各服务配置文件位于 `{service}/config-*.yaml`：

- `config-debug.yaml` - 开发环境
- `config-pro.yaml` - 生产环境

### 启动服务

```bash
# 启动用户服务
cd user && go run main.go

# 启动商品服务
cd goods && go run main.go

# 启动库存服务
cd inventory && go run main.go

# 启动订单服务
cd order && go run main.go
```

---

## 📄 协议

本项目采用 MIT 协议开源。
