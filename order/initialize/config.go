package initialize

import (
	"encoding/json"
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"order/global"
	"os"
)

func GetEnvInfo(env string) bool {
	viper.AutomaticEnv()
	return viper.GetBool(env)
}

func InitConfig() {
	data := GetEnvInfo("MALL_DEBUG")
	var configFileName string
	configFileNamePreFix := "config"
	path, _ := os.Getwd()
	if data {
		configFileName = fmt.Sprintf("%s/%s-debug.yaml", path, configFileNamePreFix)
	} else {
		configFileName = fmt.Sprintf("%s/%s-pro.yaml", path, configFileNamePreFix)
	}
	fmt.Println("configFileName", configFileName)
	v := viper.New()
	v.SetConfigFile(configFileName)
	if err := v.ReadInConfig(); err != nil {
		panic(err)
	}
	err := v.Unmarshal(global.NacosConfig)
	if err != nil {
		panic(err)
	}
	//nacos中读取配置信息
	//https://github.com/nacos-group/nacos-sdk-go/blob/master/README_CN.md
	sc := []constant.ServerConfig{
		{
			IpAddr: global.NacosConfig.Host,
			Port:   global.NacosConfig.Port,
		},
	}
	cc := constant.ClientConfig{
		NamespaceId:         global.NacosConfig.Namespace, // 如果需要支持多namespace，我们可以场景多个client,它们有不同的NamespaceId
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              "tmp/nacos/log",
		CacheDir:            "tmp/nacos/cache",
		LogLevel:            "debug",
		Username:            global.NacosConfig.User,
		Password:            global.NacosConfig.Password,
	}
	configClient, err := clients.CreateConfigClient(map[string]interface{}{
		"serverConfigs": sc,
		"clientConfig":  cc,
	})
	if err != nil {
		panic(err)
	}
	content, err := configClient.GetConfig(vo.ConfigParam{
		DataId: global.NacosConfig.DataId,
		Group:  global.NacosConfig.Group})
	if err != nil {
		panic(err)
	}
	//绑定配置中心文件
	err = json.Unmarshal([]byte(content), &global.ServerConfig)
	if err != nil {
		zap.S().Fatalf("配置文件读取失败： %s", err.Error())
	}
	fmt.Println(global.ServerConfig)
}
