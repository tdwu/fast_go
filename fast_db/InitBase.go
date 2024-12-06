package fast_db

import (
	"fmt"
	"gorm.io/gorm"
	"time"
)

var ConfigDataSource = DataSourceConfig{Enable: true, LogLevel: "info", Host: "127.0.0.1", Port: "3306", DriverName: "mysql", Params: "charset=utf8mb4&parseTime=true", MaxIdleConns: 5, MaxOpenConns: 100, MaxIdleTime: 0, ConnMaxLifetime: 60 * 60}
var ConfigSnowWorker = SnowWorkerConfig{WorkId: 0, CenterId: 0}

var SnowMaker *SnowWorker
var DB *gorm.DB

type DataSourceConfig struct {
	Enable          bool   `yaml:"enable"`
	DriverName      string `yaml:"driverName"`
	Host            string `yaml:"host"`
	Port            string `yaml:"port"`
	Database        string `yaml:"database"`
	Username        string `yaml:"username"`
	Password        string `yaml:"password"`
	Params          string `yaml:"params"`
	MaxIdleConns    int
	MaxOpenConns    int
	MaxIdleTime     time.Duration // 单位秒
	ConnMaxLifetime time.Duration // 单位秒
	LogLevel        string        // 日志打印级别 debug  info  warning  error
}

func (t DataSourceConfig) DNS() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?%s", t.Username, t.Password, t.Host, t.Port, t.Database, t.Params)
}

type SnowWorkerConfig struct {
	WorkId   int64
	CenterId int64
}
