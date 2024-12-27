package fast_base

import (
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger 日志
var Logger *zap.Logger
var LoggerLevel zapcore.Level

// ConfigAll 存储所有配置
var ConfigAll *viper.Viper

// ConfigLog 日志相关配置
var ConfigLog = LogConfig{Level: "info", Format: "", Path: ExecPath() + "/logs/", FileName: "bill.log", FileMaxSize: 10, FileMaxBackups: 100, MaxAge: 30, Compress: true, Stdout: true, Color: true}

// ConfigEnv 多环境先关配置
var ConfigEnv = EnvConfig{Env: "dev", Name: "tpl"}

type LogConfig struct {
	Level          string // 日志打印级别 debug  info  warning  error
	Format         string // 输出日志格式	logFormat, json
	Path           string // 输出日志文件路径
	FileName       string // 输出日志文件名称
	FileMaxSize    int    // 【日志分割】单个日志文件最多存储量 单位(mb)
	FileMaxBackups int    // 【日志分割】日志备份文件最多数量
	MaxAge         int    // 日志保留时间，单位: 天 (day)
	Compress       bool   // 是否压缩日志
	Stdout         bool   // 是否输出到控制台
	Color          bool   // 日志打印, 是否显示颜色
}

var LogLevelMap = map[string]zapcore.Level{
	"debug": zapcore.DebugLevel,
	"info":  zapcore.InfoLevel,
	"warn":  zapcore.WarnLevel,
	"error": zapcore.ErrorLevel,
}

type EnvConfig struct {
	Env  string
	Name string
}

func (a EnvConfig) GetApplicationId() string {
	return a.Name + "_" + a.Env
}
