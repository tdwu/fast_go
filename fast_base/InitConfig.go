package fast_base

import (
	"flag"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/viper"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// viper支持从多个数据源读取配置值，因此当同一个配置key在多个数据源有值时，viper读取的优先级如下：
// 1 显示使用Set函数设置值
// 2 flag：命令行参数
// 3 env：环境变量
// 4 config：配置文件
// 5 key/value store：key/value存储系统，如(etcd)
// 6 default:默认值

// LoadConfig 加载配置信息
func LoadConfig() (err error) {
	// 1 加载默认配置文件
	allInOne, err := loadYaml("application")
	// 注：需要通过set，写入override，否则下面的env无法合并。
	keys := allInOne.AllKeys()
	for i := range keys {
		k := keys[i]
		v := allInOne.Get(k)
		allInOne.Set(k, v)
	}

	env := getEnv(allInOne, "")
	// 2 加载多环境的配置
	if env != "" {
		tv, _ := loadYaml("application-" + env)
		keys := tv.AllKeys()
		for i := range keys {
			k := keys[i]
			v := tv.Get(k)
			allInOne.Set(k, v)
		}
	}

	ConfigAll = allInOne

	// 设置json扩展器
	jsoniter.RegisterExtension(&JsonExtension{})
	return
}

func loadYaml(name string) (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigName(name)
	v.SetConfigType("yaml")
	v.AddConfigPath("./conf")
	v.AddConfigPath(".")
	v.AddConfigPath(ExecPath())
	v.AddConfigPath(path.Join(ExecPath(), "/conf"))
	v.AddConfigPath(path.Join(ExecPath(), "/bin/conf"))
	e := v.ReadInConfig()
	return v, e
}

func getEnv(v *viper.Viper, dft string) string {
	env := ""
	// 1 命令行优先级最高
	flag.StringVar(&env, "env", "", "env active profile")
	flag.Parse()

	// 2 其次是环境变量
	if len(env) == 0 {
		env = os.Getenv("GO_ENV")
	}

	// 3 再是默认配置文件
	if len(env) == 0 {
		env = v.GetString("env")
	}

	// 4 最后是默认
	if len(env) == 0 {
		env = dft
	}

	return env
}

func IfStr(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

var ep = ""

// ExecPath 获取可执行文件的绝对路径
func ExecPath() string {
	if ep == "" {
		file, _ := os.Executable()
		path := filepath.Dir(file)
		path = strings.ReplaceAll(path, "\\", "/")
		ep = path
	}
	return ep
}
