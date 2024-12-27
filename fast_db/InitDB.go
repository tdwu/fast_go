package fast_db

import (
	"context"
	"errors"
	"fmt"
	"github.com/tdwu/fast_go/fast_base"
	"go.uber.org/zap/zapcore"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"runtime"
	"strings"
	"time"
)

// LoadDataSource 包初始化函数，golang特性，每个包初始化的时候会自动执行init函数，这里用来初始化gorm。
func LoadDataSource() {

	fast_base.ConfigAll.UnmarshalKey("dataSource", &ConfigDataSource)
	fast_base.ConfigAll.UnmarshalKey("snowWorker", &ConfigSnowWorker)

	if !ConfigDataSource.Enable {
		fast_base.Logger.Info("数据库 未启用")
		return
	}

	// 数据库版本管理工具
	migrateDB()

	// 启用雪花算法
	SnowMaker = NewSnowWorker(ConfigSnowWorker.WorkId)

	// 连接MYSQL, 获得DB类型实例，用于后面的数据库读写操作。
	level, ok := fast_base.LogLevelMap[ConfigDataSource.LogLevel] // 日志打印级别
	if !ok {
		level = fast_base.LogLevelMap["info"]
	}

	// 获得一个*grom.DB对象
	_db, err := gorm.Open(
		mysql.Open(ConfigDataSource.DNS()),
		//日志框架替换
		&gorm.Config{
			NamingStrategy: schema.NamingStrategy{SingularTable: true}, // 表名不加s.

			PrepareStmt: true,
			//	Strict: true,
			// 定制化logger,与zap日志框架集成
			Logger: customGormLogger(logger.Config{
				SlowThreshold: 200 * time.Millisecond, //设定慢查询时间阈值为1ms
				//设置日志级别，只有Warn和Info级别会输出慢查询日志
				//LogLevel:                  logger.Warn,
				IgnoreRecordNotFoundError: false,
				Colorful:                  fast_base.ConfigLog.Color,
			}, level),
		})
	if err != nil {
		panic("连接数据库失败, error=" + err.Error())
	}

	// 设置为全局
	DB = _db

	// 连接池配置(Go DataBase Api 自带了连接池)
	sqlDB, _ := _db.DB()

	//连接池最多同时打开的连接数。
	//这个maxOpenConns理应要设置得比mysql服务器的max_connections值要小。
	//一般设置为： 服务器cpu核心数 * 2 + 服务器有效磁盘数。参考这里
	//可用show variables like ‘max_connections’; 查看服务器当前设置的最大连接数。
	sqlDB.SetMaxOpenConns(ConfigDataSource.MaxOpenConns)

	//连接池最大允许的空闲连接数。必须要比maxOpenConns小，超过的连接会被连接池关闭。
	sqlDB.SetMaxIdleConns(ConfigDataSource.MaxIdleConns)

	//连接池里面的连接最大空闲时长。
	//当连接持续空闲时长达到maxIdleTime后，该连接就会被关闭并从连接池移除，【注意】哪怕当前空闲连接数已经小于SetMaxIdleConns(maxIdleConns)设置的值。
	//连接每次被使用后，持续空闲时长会被重置，从0开始从新计算；
	//用show processlist; 可用查看mysql服务器上的连接信息，Command表示连接的当前状态，Command为Sleep时表示休眠、空闲状态，Time表示此状态的已持续时长；
	//建议设置为0，不启用
	sqlDB.SetConnMaxIdleTime(ConfigDataSource.MaxIdleTime * time.Second)

	//连接池里面的连接最大存活时长。
	//maxLifeTime必须要比mysql服务器设置的wait_timeout小，否则会导致golang侧连接池依然保留已被mysql服务器关闭了的连接。
	//mysql服务器的wait_timeout默认是8 hour，可通过show variables like 'wait_timeout’查看。
	sqlDB.SetConnMaxLifetime(ConfigDataSource.ConnMaxLifetime * time.Second)

	// 【问题】没心跳sql？

	fast_base.DictQueryBySql = func(sql string, p ...interface{}) string {
		var v string
		err = DB.Raw(sql, p...).Scan(&v).Error
		if err != nil {
			return err.Error()
		}
		return v
	}
}

type Model struct {
	ID        fast_base.StringInt64 `gorm:"primarykey" `
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// //////////////////////////////////定制化日志器//////////////////////////////////////////////////////////
func customGormLogger(config logger.Config, level zapcore.Level) logger.Interface {
	var (
		infoStr      = "%s\n[info] "
		warnStr      = "%s\n[warn] "
		errStr       = "%s\n[error] "
		traceStr     = "[%.3fms] [rows:%v] %s"
		traceWarnStr = "%s\n[%.3fms] [rows:%v] %s"
		traceErrStr  = "%s\n[%.3fms] [rows:%v] %s"
	)

	if config.Colorful {
		infoStr = logger.Green + "%s\n" + logger.Reset + logger.Green + "[info] " + logger.Reset
		warnStr = logger.BlueBold + "%s\n" + logger.Reset + logger.Magenta + "[warn] " + logger.Reset
		errStr = logger.Magenta + "%s\n" + logger.Reset + logger.Red + "[error] " + logger.Reset
		traceStr = logger.Reset + logger.Yellow + "[%.3fms] " + logger.BlueBold + "[rows:%v]" + logger.Reset + " %s"
		traceWarnStr = logger.Yellow + "%s\n" + logger.Reset + logger.RedBold + "[%.3fms] " + logger.Yellow + "[rows:%v]" + logger.Magenta + " %s" + logger.Reset
		traceErrStr = logger.MagentaBold + "%s\n" + logger.Reset + logger.Yellow + "[%.3fms] " + logger.BlueBold + "[rows:%v]" + logger.Reset + " %s"
	}
	// 设置日志级别
	config.LogLevel = convertToDbLogLevel(level)
	return &GormLogger{
		Config:       config,
		infoStr:      infoStr,
		warnStr:      warnStr,
		errStr:       errStr,
		traceStr:     traceStr,
		traceWarnStr: traceWarnStr,
		traceErrStr:  traceErrStr,
	}
}
func convertToDbLogLevel(l zapcore.Level) logger.LogLevel {
	if l >= zapcore.DPanicLevel {
		return logger.Silent
	} else if l >= zapcore.ErrorLevel {
		return logger.Error
	} else if l >= zapcore.WarnLevel {
		return logger.Warn
	} else {
		return logger.Info
	}
}

// GormLogger //////////////////////////////////日志器接口的实现//////////////////////////////////////////////////////////
type GormLogger struct {
	ZapLevel zapcore.Level
	logger.Config
	infoStr, warnStr, errStr            string
	traceStr, traceErrStr, traceWarnStr string
}

// LogMode log mode
func (l *GormLogger) LogMode(level logger.LogLevel) logger.Interface {
	newlogger := *l
	newlogger.LogLevel = level
	return &newlogger
}

// Info print info
func (l GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Info {
		fast_base.PrintfWithCaller(l.ZapLevel, findGormCaller(), msg, data...)
	}
}

// Warn print warn messages
func (l GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Warn {
		fast_base.PrintfWithCaller(l.ZapLevel, findGormCaller(), msg, data...)
	}
}

// Error print error messages
func (l GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Error {
		fast_base.PrintfWithCaller(l.ZapLevel, findGormCaller(), msg, data...)
	}
}

// Trace print sql message
func (l GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil && l.LogLevel >= logger.Error && (!errors.Is(err, logger.ErrRecordNotFound) || !l.IgnoreRecordNotFoundError):
		sql, rows := fc()
		if rows == -1 {
			fast_base.PrintfWithCaller(l.ZapLevel, findGormCaller(), l.traceErrStr, err, float64(elapsed.Nanoseconds())/1e6, "-", sql)
			//	l.Printf(l.traceErrStr, utils.FileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			fast_base.PrintfWithCaller(l.ZapLevel, findGormCaller(), l.traceErrStr, err, float64(elapsed.Nanoseconds())/1e6, rows, sql)
			//	l.Printf(l.traceErrStr, utils.FileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= logger.Warn:
		sql, rows := fc()
		slowLog := fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold)
		if rows == -1 {
			fast_base.PrintfWithCaller(l.ZapLevel, findGormCaller(), l.traceWarnStr, slowLog, float64(elapsed.Nanoseconds())/1e6, "-", sql)
			//l.Printf(l.traceWarnStr, utils.FileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			fast_base.PrintfWithCaller(l.ZapLevel, findGormCaller(), l.traceWarnStr, slowLog, float64(elapsed.Nanoseconds())/1e6, rows, sql)
			//l.Printf(l.traceWarnStr, utils.FileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case l.LogLevel == logger.Info:
		sql, rows := fc()
		if rows == -1 {
			fast_base.PrintfWithCaller(l.ZapLevel, findGormCaller(), l.traceStr, float64(elapsed.Nanoseconds())/1e6, "-", sql)
			//l.Printf(l.traceStr, utils.FileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			fast_base.PrintfWithCaller(l.ZapLevel, findGormCaller(), l.traceStr, float64(elapsed.Nanoseconds())/1e6, rows, sql)
			//l.Printf(l.traceStr, utils.FileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	}
}

func findGormCaller() *zapcore.EntryCaller {
	for i := 2; i < 15; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if ok && (!strings.Contains(file, "gorm.io") || strings.HasSuffix(file, "_test.go")) {
			//fmt.Println("--- " + file + " :" + strconv.Itoa(line))
			return &zapcore.EntryCaller{Defined: true, PC: pc, File: file, Line: line, Function: runtime.FuncForPC(pc).Name()}
			//return file + ":" + strconv.FormatInt(int64(line), 10) + "" + runtime.FuncForPC(pc).Name()
		}
	}
	return nil
}
