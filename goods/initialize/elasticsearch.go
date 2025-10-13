package initialize

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/elastic/elastic-transport-go/v8/elastictransport"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"go.uber.org/zap"
	"goods/global"
	"goods/model"
	"net/http"
	"os"
	"strings"
)

func InitElasticsearch() {
	//连接elasticsearch服务
	address := fmt.Sprintf("https://%s:%d", global.ServerConfig.ElasticsearchInfo.Host, global.ServerConfig.ElasticsearchInfo.Port)
	cfg := elasticsearch.Config{

		Addresses: []string{address},
		Username:  global.ServerConfig.ElasticsearchInfo.Username,
		Password:  global.ServerConfig.ElasticsearchInfo.Password,
		Logger: &elastictransport.ColorLogger{
			Output:             os.Stdout,
			EnableRequestBody:  true,
			EnableResponseBody: true,
		},
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // 跳过证书验证（仅测试环境）
		},
	}
	var err error
	global.EsClient, err = elasticsearch.NewClient(cfg)
	if err != nil {
		zap.S().Panicf("初始化连接Elasticsearch失败: %s", err)
	}
	res, err := global.EsClient.Info()
	if err != nil {
		zap.S().Errorf("Elasticsearch获取响应时出错: %s", err)
	}
	defer res.Body.Close()
	// 检查索引是否存在
	exists, err := global.EsClient.Indices.Exists([]string{model.EsGoods{}.GetIndexName()})
	if err != nil {
		zap.S().Panicf("Error checking index existence: %s", err)
	}
	exists.Body.Close()
	if exists.StatusCode != 200 {
		req := esapi.IndicesCreateRequest{
			Index: model.EsGoods{}.GetIndexName(),
			Body:  strings.NewReader(model.EsGoods{}.GetMappings()),
		}
		res, err = req.Do(context.Background(), global.EsClient)
		if err != nil {
			zap.S().Panicf("创建索引失败: %s", err)
		}
	}

}
