package global

import (
	"github.com/go-redsync/redsync/v4"
	"gorm.io/gorm"
	"inventory/config"
)

var (
	DB           *gorm.DB
	Rs           *redsync.Redsync
	ServerConfig config.ServerConfig
	NacosConfig  *config.NacosConfig = &config.NacosConfig{}
)
