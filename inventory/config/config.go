package config

type MysqlConfig struct {
	Host     string `mapstructure:"host" json:"host"`
	Port     int    `mapstructure:"port" json:"port"`
	Db       string `mapstructure:"db" json:"db"`
	User     string `mapstructure:"user" json:"user"`
	Password string `mapstructure:"password" json:"password"`
}

type ConsulConfig struct {
	Host string `mapstructure:"host" json:"host"`
	Port int    `mapstructure:"port" json:"port"`
}

type NacosConfig struct {
	Host      string `mapstructure:"host"`
	Port      uint64 `mapstructure:"port"`
	Namespace string `mapstructure:"namespace"`
	User      string `mapstructure:"user"`
	Password  string `mapstructure:"password"`
	DataId    string `mapstructure:"dataid"`
	Group     string `mapstructure:"group"`
}
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     uint64 `mapstructure:"port"`
	Db       int    `mapstructure:"db"`
	Password string `mapstructure:"password"`
}
type ServerConfig struct {
	Name       string       `mapstructure:"name" json:"name"`
	Tags       []string     `mapstructure:"tags" json:"tags"`
	Host       string       `mapstructure:"host" json:"host"`
	MysqlInfo  MysqlConfig  `mapstructure:"mysql" json:"mysql"`
	ConsulInfo ConsulConfig `mapstructure:"consul" json:"consul"`
	NacosInfo  NacosConfig  `mapstructure:"nacos" json:"nacos"`
	RedisInfo  RedisConfig  `mapstructure:"redis" json:"redis"`
	MqInfo     MqConfig     `mapstructure:"mq_config" json:"mq_config"`
}
type MqConfig struct {
	Host         string `mapstructure:"host"  json:"host"`
	Port         uint64 `mapstructure:"port"  json:"port"`
	InvGroupName string `mapstructure:"inv_group_name"  json:"inv_group_name"`
}
