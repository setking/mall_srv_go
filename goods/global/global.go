package global

import (
	"github.com/elastic/go-elasticsearch/v8"
	"goods/config"
	"gorm.io/gorm"
)

var (
	DB           *gorm.DB
	ServerConfig config.ServerConfig
	NacosConfig  *config.NacosConfig = &config.NacosConfig{}
	EsClient     *elasticsearch.Client
)
