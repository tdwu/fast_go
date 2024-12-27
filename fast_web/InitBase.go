package fast_web

import (
	"fmt"
)

var ConfigServer = ServerConfig{LogLevel: "debug", Host: "0.0.0.0", Port: "8080", Upload: "./uploadStore/",
	Static: &ServerStaticConfig{
		Root: "${execPath}/web/ui",
	},
	Session: &Session{
		Duration: "60",
	},
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`

	Template string
	Static   *ServerStaticConfig
	Session  *Session
	Upload   string
	LogLevel string // 日志打印级别 debug  info  warning  error
}

type ServerStaticConfig struct {
	Root         string
	PathPatterns []string
	ResLocations []string
}

type Session struct {
	Duration string `yaml:"duration"`
}

func (t ServerConfig) Address() string {
	return fmt.Sprintf("%s:%s", t.Host, t.Port)
}

// ApiGroup
// Deprecated 统一换成使用gr生成
type ApiGroup struct {
	Parent string
}
