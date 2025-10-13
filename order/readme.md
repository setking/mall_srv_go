### proto文件编译命令：
```azure
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative user.proto
```

### 安装elasticsearch
#### 新建es的config配置⽂件夹
```azure
 mkdir -p /data/elasticsearch/config
```
#### 新建es的data⽬录
```azure
 mkdir -p /data/elasticsearch/data
```

#### 新建es的plugins⽬录
```azure
 mkdir -p /data/elasticsearch/plugins
```
#### 给⽬录设置权限
```azure
 chmod 777 -R /data/elasticsearch
```
#### 写⼊配置到elasticsearch.yml中， 下⾯的 > 表示覆盖的⽅式写⼊， >>表示追加的⽅式写⼊，但是要确
```azure
 echo "http.host: 0.0.0.0" >> /data/elasticsearch/config/elasticsearch.yml
```
#### 安装es
```azure
docker run --name elasticsearch -p 9200:9200 -p 9300:9300 -e "discovery.type=single-node"  -e ES_JAVA_OPTS="-Xms256m -Xmx512m" -v /data/elasticsearch/config/elasticsearch.yml:/usr/share/elasticsearch/config/elasticsearch.yml -v /data/elasticsearch/data:/usr/share/elasticsearch/data -v /data/elasticsearch/plugins:/usr/share/elasticsearch/plugins -d elasticsearch:8.19.0
```
#### 安装kibana
```azure
docker run -d --name kibana -e ELASTICSEARCH="http://192.168.0.109:9200" -p 5601:5601 kibana:8.19.0
```



docker run -d \
--restart=always \
--name elasticsearch \
--network host \
-p 9388:9200 \
-p 9389:9300 \
--privileged \
-e "discovery.type=single-node" \
-e "ES_JAVA_OPTS=-Xms256m -Xms512m" \
docker.elastic.co/elasticsearch/elasticsearch:8.19.0


docker run -d \
--name elasticsearch \
--restart=always \
--network host \
-v "/data/elasticsearch/data:/usr/share/elasticsearch/data" \
-v "/data/elasticsearch/plugins:/usr/share/elasticsearch/plugins" \
-v "/data/elasticsearch/config:/usr/share/elasticsearch/config" \
-v "/data/elasticsearch/logs:/usr/share/elasticsearch/logs" \
-e "discovery.type=single-node" \
-e "ES_JAVA_OPTS=-Xms256m -Xms512m" \
-e "ELASTIC_PASSWORD=123456" \
docker.elastic.co/elasticsearch/elasticsearch:8.19.0