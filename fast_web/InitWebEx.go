package fast_web

import (
	"bytes"
	"errors"
	"fast_base"
	"fmt"
	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap/zapcore"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"
)

type LogWriter struct {
	level zapcore.Level
}

func (l LogWriter) Write(p []byte) (n int, err error) {
	var message string
	// 去掉最后一个换行符，本身日志框架已经自带了换行符
	if len(p) > 0 && p[len(p)-1] == '\n' {
		message = string(p[:len(p)-1])
	} else {
		message = string(p)
	}

	fast_base.PrintfWithCaller(l.level, findGinCaller(4), message)

	return 0, nil
}

func findGinCaller(s int) *zapcore.EntryCaller {
	for i := s; i < 15; i++ {
		pc, file, line, ok := runtime.Caller(i)
		function := runtime.FuncForPC(pc).Name()
		//fmt.Println("--- " + file + " :" + strconv.Itoa(line) + "  " + runtime.FuncForPC(pc).Name())
		if ok && (!strings.Contains(function, "debugPrint")) {
			return &zapcore.EntryCaller{Defined: true, PC: pc, File: file, Line: line, Function: runtime.FuncForPC(pc).Name()}
			//return file + ":" + strconv.FormatInt(int64(line), 10) + "" + runtime.FuncForPC(pc).Name()
		}
	}
	return nil
}

// //////////////////////////////////为gin创建日志中间件，集成zap日志框架//////////////////////////////////////////////////////////
func ginLogger() gin.HandlerFunc {
	var skip map[string]struct{}

	level, ok := fast_base.LogLevelMap[ConfigServer.LogLevel] // 日志打印级别
	if !ok {
		level = fast_base.LogLevelMap["info"]
	}

	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// 日志级别，比同一的日志级别小
		if !fast_base.LoggerLevel.Enabled(level) {
			return
		}

		// Log only when path is not being skipped
		if _, ok := skip[path]; !ok {
			param := gin.LogFormatterParams{
				Request: c.Request,

				Keys: c.Keys,
			}

			// Stop timer
			param.TimeStamp = start
			param.Latency = time.Now().Sub(start)

			param.ClientIP = c.ClientIP()
			param.Method = c.Request.Method
			param.StatusCode = c.Writer.Status()
			param.ErrorMessage = c.Errors.ByType(gin.ErrorTypePrivate).String()

			param.BodySize = c.Writer.Size()

			if raw != "" {
				//编码
				//escapeUrl := url.QueryEscape(raw)
				//解码
				enEscapeUrl, _ := url.QueryUnescape(raw)

				path = path + "?" + enEscapeUrl
			}

			param.Path = path

			message := formatMessage(param)

			fast_base.PrintfWithCaller(level, findGinCaller(0), message)
		}
	}
}

func formatMessage(param gin.LogFormatterParams) string {
	var statusColor, methodColor, resetColor string
	if fast_base.ConfigLog.Color {
		statusColor = param.StatusCodeColor()
		methodColor = param.MethodColor()
		resetColor = param.ResetColor()
	}

	if param.Latency > time.Minute {
		param.Latency = param.Latency.Truncate(time.Second)
	}
	if len(param.ErrorMessage) == 0 {
		return fmt.Sprintf("[Monitor] %v |%s %3d %s| %13v | %15s |%s %-7s %s %#v",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			// status color
			statusColor, param.StatusCode, resetColor,
			param.Latency,
			param.ClientIP,
			// method color
			methodColor, param.Method, resetColor,
			param.Path,
		)
	} else {
		return fmt.Sprintf("[Monitor] %v |%s %3d %s| %13v | %15s |%s %-7s %s %#v \n%s",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			// status color
			statusColor, param.StatusCode, resetColor,
			param.Latency,
			param.ClientIP,
			// method color
			methodColor, param.Method, resetColor,
			param.Path,
			param.ErrorMessage,
		)
	}
}

// //////////////////////////////////为gin创建全局异常中间件，异常适用zap日志框架打印//////////////////////////////////////////////////////////
// recover掉项目可能出现的panic
func ginRecovery() gin.HandlerFunc {
	return customRecover(func(c *gin.Context, err any) {
		//	c.AbortWithStatus(http.StatusInternalServerError)
		c.JSON(http.StatusBadRequest, fast_base.Error(501, "未知错误", err))
	})
}

var (
	dunno     = []byte("???")
	centerDot = []byte("·")
	dot       = []byte(".")
	slash     = []byte("/")
)

const (
	green   = "\033[97;42m"
	white   = "\033[90;47m"
	yellow  = "\033[90;43m"
	red     = "\033[97;41m"
	blue    = "\033[97;44m"
	magenta = "\033[97;45m"
	cyan    = "\033[97;46m"
	reset   = "\033[0m"
)

// CustomRecoveryWithWriter returns a middleware for a given writer that recovers from any panics and calls the provided handle func to handle it.
func customRecover(handle gin.RecoveryFunc) gin.HandlerFunc {

	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Check for a broken connection, as it is not really a
				// condition that warrants a panic stack trace.
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					var se *os.SyscallError
					if errors.As(ne, &se) {
						seStr := strings.ToLower(se.Error())
						if strings.Contains(seStr, "broken pipe") ||
							strings.Contains(seStr, "connection reset by peer") {
							brokenPipe = true
						}
					}
				}
				stack := stack(3)
				httpRequest, _ := httputil.DumpRequest(c.Request, false)
				headers := strings.Split(string(httpRequest), "\r\n")
				for idx, header := range headers {
					current := strings.Split(header, ":")
					if current[0] == "Authorization" {
						headers[idx] = current[0] + ": *"
					}
				}
				headersToStr := strings.Join(headers, "\r\n")
				if brokenPipe {
					fast_base.Logger.Error(fmt.Sprintf(fast_base.IfStr(fast_base.ConfigLog.Color, red, "")+"[Panic]: %s"+reset+"\n%s%s", err, headersToStr, reset))
				} else if gin.IsDebugging() {
					fast_base.Logger.Error(fmt.Sprintf("[Recovery] %s panic recovered:\n%s\n"+fast_base.IfStr(fast_base.ConfigLog.Color, red, "")+"[Panic]: %s"+reset+"\n%s%s",
						timeFormat(time.Now()), headersToStr, err, stack, reset))
				} else {
					fast_base.Logger.Error(fmt.Sprintf("[Recovery] %s panic recovered:\n"+fast_base.IfStr(fast_base.ConfigLog.Color, red, "")+"[Panic]:%s"+reset+"\n%s%s",
						timeFormat(time.Now()), err, stack, reset))
				}
				if brokenPipe {
					// If the connection is dead, we can't write a status to it.
					c.Error(err.(error)) //nolint: errcheck
					c.Abort()
				} else {
					handle(c, err)
				}
			}
		}()
		c.Next()
	}
}

// //////////////////////////////////由于gin自带的recovery.go方法为私有，无法扩展，只有全部复制过来修改//////////////////////////////////////////////////////////
// stack returns a nicely formatted stack frame, skipping skip frames.
func stack(skip int) []byte {
	buf := new(bytes.Buffer) // the returned data
	// As we loop, we open files and read them. These variables record the currently
	// loaded file.
	var lines [][]byte
	var lastFile string
	for i := skip; ; i++ { // Skip the expected number of frames
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		// Print this much at least.  If we can't find the source, it won't show.
		fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc)
		if file != lastFile {
			data, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			lines = bytes.Split(data, []byte{'\n'})
			lastFile = file
		}
		fmt.Fprintf(buf, "\t%s: %s\n", function(pc), source(lines, line))
	}
	return buf.Bytes()
}

// source returns a space-trimmed slice of the n'th line.
func source(lines [][]byte, n int) []byte {
	n-- // in stack trace, lines are 1-indexed but our array is 0-indexed
	if n < 0 || n >= len(lines) {
		return dunno
	}
	return bytes.TrimSpace(lines[n])
}

// function returns, if possible, the name of the function containing the PC.
func function(pc uintptr) []byte {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return dunno
	}
	name := []byte(fn.Name())
	// The name includes the path name to the package, which is unnecessary
	// since the file name is already included.  Plus, it has center dots.
	// That is, we see
	//	runtime/debug.*T·ptrmethod
	// and want
	//	*T.ptrmethod
	// Also the package path might contain dot (e.g. code.google.com/...),
	// so first eliminate the path prefix
	if lastSlash := bytes.LastIndex(name, slash); lastSlash >= 0 {
		name = name[lastSlash+1:]
	}
	if period := bytes.Index(name, dot); period >= 0 {
		name = name[period+1:]
	}
	name = bytes.Replace(name, centerDot, dot, -1)
	return name
}

// timeFormat returns a customized time string for logger.
func timeFormat(t time.Time) string {
	return t.Format("2006/01/02 - 15:04:05")
}

// 创建 jsoniter 配置
var json = jsoniter.ConfigCompatibleWithStandardLibrary

// 定义 jsoniter 的渲染器
type JSONIterRenderer struct {
	Data any
}

// Render 方法实现自定义的 JSON 渲染逻辑
func (r JSONIterRenderer) Render(w http.ResponseWriter) error {
	// 设置 Content-Type
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	// 使用 jsoniter 序列化数据并写入 ResponseWriter
	return json.NewEncoder(w).Encode(r.Data)
}

// WriteContentType 方法设置内容类型
func (r JSONIterRenderer) WriteContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
}

// JSON 输出方法，使用 jsoniter 渲染
func JSONIter(c *gin.Context, code int, obj any) {
	c.Render(code, JSONIterRenderer{Data: obj})
}
