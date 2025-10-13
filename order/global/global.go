package global

import (
	"gorm.io/gorm"
	"order/config"
	"order/proto"
)

var (
	DB           *gorm.DB
	ServerConfig config.ServerConfig
	NacosConfig  *config.NacosConfig = &config.NacosConfig{}

	GoodsSrvClient  proto.GoodsClient
	InventoryClient proto.InventoryClient
)
