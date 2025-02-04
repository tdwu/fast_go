package fast_base

// 安装以下依赖库
// go get -u go.uber.org/zap
// go get -u github.com/natefinch/lumberjack
import (
	"fmt"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"path/filepath"
)

// LoadLogger 初始化 log
func LoadLogger() error {

	ConfigAll.UnmarshalKey("log", &ConfigLog)

	writeSyncer, err := getLogWriter(ConfigLog) // 日志文件配置 文件位置和切割
	if err != nil {
		return err
	}

	encoder := getEncoder(ConfigLog)                    // 获取日志输出编码
	CoreLoggerLevel, ok := LogLevelMap[ConfigLog.Level] // 日志打印级别
	if !ok {
		CoreLoggerLevel = LogLevelMap["info"]
	}

	core := zapcore.NewCore(encoder, writeSyncer, CoreLoggerLevel)

	logger := zap.New(core, zap.AddCaller()) // zap.Addcaller() 输出日志打印文件和行数如： logger/logger_test.go:33
	// 1. zap.ReplaceGlobals 函数将当前初始化的 logger 替换到全局的 logger,
	// 2. 使用 logger 的时候 直接通过 zap.S().Debugf("xxx") or zap.L().Debug("xxx")
	// 3. 使用 zap.S() 和 zap.L() 提供全局锁，保证一个全局的安全访问logger的方式
	zap.ReplaceGlobals(logger)
	//zap.L().Debug("")
	//zap.S().Debugf("")
	Logger = logger

	logger.Debug(fmt.Sprintf("配置参数：%#v", ConfigLog))
	return nil
}

// getEncoder 编码器(如何写入日志)
func getEncoder(conf LogConfig) zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	//encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder   // log 时间格式 例如: 2021-09-11t20:05:54.852+0800
	encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000")
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder // 输出level序列化为全大写字符串，如 INFO DEBUG ERROR
	//encoderConfig.EncodeCaller = zapcore.FullCallerEncoder
	//encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	if conf.Format == "json" {
		return zapcore.NewJSONEncoder(encoderConfig) // 以json格式写入
	}

	/*
		encoderConfig := zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "name",
			CallerKey:      "line",
			MessageKey:     "msg",
			FunctionKey:    "func",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,//
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000"),
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.FullCallerEncoder,
			EncodeName:     zapcore.FullNameEncoder,
		}
	*/
	return zapcore.NewConsoleEncoder(encoderConfig) // 以logfmt格式写入
}

// getLogWriter 获取日志输出方式  日志文件 控制台
func getLogWriter(conf LogConfig) (zapcore.WriteSyncer, error) {

	// 判断日志路径是否存在，如果不存在就创建
	if exist := isExist(conf.Path); !exist {
		if conf.Path == "" {
			conf.Path = "./logs/out.log"
		}
		if err := os.MkdirAll(conf.Path, os.ModePerm); err != nil {
			conf.Path = "./logs/out.log"
			if err := os.MkdirAll(conf.Path, os.ModePerm); err != nil {
				return nil, err
			}
		}
	}

	// 日志文件 与 日志切割 配置
	lumberJackLogger := &lumberjack.Logger{
		Filename:   filepath.Join(conf.Path, conf.FileName), // 日志文件路径
		MaxSize:    conf.FileMaxSize,                        // 单个日志文件最大多少 mb
		MaxBackups: conf.FileMaxBackups,                     // 日志备份数量
		MaxAge:     conf.MaxAge,                             // 日志最长保留时间
		LocalTime:  true,                                    // 本地时区
		Compress:   conf.Compress,                           // 是否压缩日志
	}
	if conf.Stdout {
		// 日志同时输出到控制台和日志文件中
		return zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(lumberJackLogger)), nil
	} else {
		// 日志只输出到日志文件
		return zapcore.AddSync(lumberJackLogger), nil
	}
}

// isExist 判断文件或者目录是否存在
func isExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func PrintfWithCaller(level zapcore.Level, caller *zapcore.EntryCaller, format string, a ...interface{}) {
	message := fmt.Sprintf(format, a...)
	if ce := Logger.Check(level, message); ce != nil {
		if caller != nil {
			ce.Entry.Caller = *caller
		}
		ce.Write()
	}
}

func PrintfWithoutCaller(level zapcore.Level, format string, a ...interface{}) {
	message := fmt.Sprintf(format, a...)
	if ce := Logger.Check(level, message); ce != nil {
		ce.Write()
	}
}

func LogInfo(format string, a ...interface{}) {
	Logger.Info(fmt.Sprintf(format, a...))
}

func LogDebug(format string, a ...interface{}) {
	Logger.Debug(fmt.Sprintf(format, a...))

}

func LogError(format string, a ...interface{}) {
	Logger.Error(fmt.Sprintf(format, a...))
}
