package fast_web

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/tdwu/fast_go/fast_base"
	"github.com/tdwu/fast_go/fast_web/web/proxy"
	"go.uber.org/zap/zapcore"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"
)

var operator = map[string]string{"post": "POST" /*"save": "POST", */, "update": "POST", "upload": "POST", "delete": "POST"}

// type-->new-->value
// value-->Type()-->type

// value
//    +type    func(web.CGroup, *gin.Context, web.User, string)
//    +ptr     <func(*gin.Context, web.User, string) Value> //汇编中类的方法调用第一个参数会把类[本身指针]作为[第一个参数]入栈

// userPtrValue.Elem()  指针类型value，变成值类型value
// userValue.Addr()     值类型value，变成指针类型value

// tm.Func.String()=        <func(config.CGroup, *gin.Context) Value>
// tm.Func.Type().String()= func(config.CGroup, *gin.Context)
// tm.Func.Type().NumIn()=2
// tm.Type.String()=         func(config.CGroup, *gin.Context)
// tm.Type.NumIn()=    2
// vm.Type().String()=       func(*gin.Context) 根据value来的，有实例接受者，所以少一个参数
// vm.Type().NumIn()=  1
// vm.String()=              <func(*gin.Context) Value>
// 汇编中类的方法调用第一个参数会把类[本身指针]作为[第一个参数]入栈, 然后再入栈其它参数,这里同cpp的汇编调用方式
// 因为方法里可能会用到里面的字段 所以需要该结构体的内存首地址

func LoadWebAll() *Server {
	fast_base.LoadConfig()
	fast_base.LoadLogger()
	LoadValidator()
	return LoadWeb()
}

func LoadWeb() *Server {

	fast_base.ConfigAll.UnmarshalKey("server", &ConfigServer)
	//gin.SetMode("release")
	gin.DefaultWriter = LogWriter{level: fast_base.LoggerLevel}
	gin.DefaultErrorWriter = LogWriter{level: zapcore.ErrorLevel}

	d, _ := os.Getwd()
	fast_base.Logger.Info("启动目录：" + d)
	execPath := strings.ReplaceAll(fast_base.ExecPath(), "\\", "/")
	fast_base.Logger.Info("程序目录：" + d)

	Container = new(Server)
	Container.Gin = gin.New()

	// 日志中间件
	Container.Gin.Use(ginLogger(), ginRecovery())

	// 跨域配置
	allowCross := fast_base.ConfigAll.GetBool("server.cross.allow")
	if allowCross == true {
		// 允许跨域
		Container.Gin.Use(CORSMiddleware())
	}

	//root := strings.Replace(ConfigServer.Static.Root, "${execPath}", execPath, -1)
	//默认首页
	//Container.Gin.StaticFile("/", root+"/index.html")
	//Container.Gin.StaticFile("/index.html", root+"/index.html")
	//Container.Gin.StaticFile("/favicon.ico", root+"/favicon.ico")

	// 静态资源
	for i, pattern := range ConfigServer.Static.PathPatterns {
		if i < len(ConfigServer.Static.ResLocations) {
			fl := strings.Replace(ConfigServer.Static.ResLocations[i], "${execPath}", execPath, -1)
			Container.LoadStatic(pattern, fl)
			fast_base.Logger.Info("静态资源[" + pattern + "]-->" + fl)
		} else {
			fast_base.Logger.Error("静态资源[" + pattern + "],未找到对应的路径")
		}
	}

	// 模板
	Container.Gin.Delims("{%", "%}")
	if ConfigServer.Template != "" {
		fl := strings.Replace(ConfigServer.Template, "${execPath}", execPath, -1)
		fast_base.Logger.Info("模版[" + fl + "]")
		Container.Gin.LoadHTMLGlob(fl)
	}

	Container.Gin.GET("shutdown", func(context *gin.Context) {
		fast_base.Logger.Warn("收到指令关闭")
		JSONIter(context, http.StatusForbidden, fast_base.Success("预计2S后完成,关闭中....."))
		context.Abort()
		go func() {
			time.Sleep(time.Second * 2)
			fast_base.Logger.Warn("执行关闭")
			os.Exit(0)
		}()
	})

	return Container
}

type HandlerFunc func(*gin.Engine)

var Container *Server

type Server struct {
	Gin        *gin.Engine
	HttpServer *http.Server
}

func (c *Server) LoadRouters(handlerFunc HandlerFunc) *Server {
	handlerFunc(c.Gin)
	return c
}

func (c *Server) Stop() *Server {

	return c
}

// LoadRouter 加载ApiGroup下的API
// Deprecated 统一换成使用gr生成
func (c *Server) LoadRouter(cg interface{}) *Server {
	engine := Container.Gin
	value := reflect.ValueOf(cg)
	numOfMethod := value.Type().NumMethod()
	//PrintfLog(logLevel, "[GIN-"+gin.Mode()+"] LoadRouter: "+value.Type().String())
	fast_base.PrintfWithCaller(fast_base.LoggerLevel, findGinCaller(1), "[GIN-"+gin.Mode()+"] LoadRouter: "+value.Type().String())

	for i := 0; i < numOfMethod; i++ {
		tm := value.Type().Method(i)
		vm := value.MethodByName(tm.Name)

		//PrintfLog(logLevel, " - method name:%s ,func:%s, type:%s, str:%#v", tm.Name, tm.Func, tm.Type, tm)
		//PrintfLog(logLevel, " - method kind:%s ,type:%s, str:%#v", vm.Kind(), tm.Type, tm)

		// http method
		mName := strings.ToLower(tm.Name)
		var em = reflect.ValueOf(engine).MethodByName("GET")
		for k, v := range operator {
			if strings.HasPrefix(mName, k) {
				em = reflect.ValueOf(engine).MethodByName(v)
				break
			}
		}

		// 为ApiGroup实现http api path，后面统一换成
		path := tm.Name
		parent := value.FieldByName("Parent")
		if parent.IsValid() {
			pStr := parent.Interface().(string)
			if strings.HasSuffix(pStr, "/") {
				path = pStr + path
			} else {
				path = pStr + "/" + path
			}
		}

		// 只有一个参数无返回值，并且是gin.Context的情况，说明是gin原型：func(*gin.Context)
		if vm.Type().NumOut() == 0 && vm.Type().NumIn() == 1 && strings.HasSuffix(vm.Type().In(0).String(), "gin.Context") {
			// 注册路由，使用handlerFunc原型
			em.Call([]reflect.Value{reflect.ValueOf(path), vm})
			continue
		}

		// 封装一层
		handlerFunc := HandlerFuncWrapper(vm)

		// 注册路由,包装封装一层，提供参数处理
		em.Call([]reflect.Value{reflect.ValueOf(path), reflect.ValueOf(handlerFunc)})
	}
	return c
}

func GenHandlerFunc(vm reflect.Value) gin.HandlerFunc {
	// 只有一个参数无返回值，并且是gin.Context的情况:
	// 说明是gin原型：func(*gin.Context)则不处理.预留给下载文件等使用
	if vm.Type().NumOut() == 0 && vm.Type().NumIn() == 1 && strings.HasSuffix(vm.Type().In(0).String(), "gin.Context") {
		//p := vm.Interface().(gin.HandlerFunc) // 不能用vm.Interface().(gin.HandlerFunc)
		p := vm.Interface().(func(*gin.Context))
		return p
	}
	// 封装一层
	return HandlerFuncWrapper(vm)
}

func HandlerFuncWrapper(vm reflect.Value) gin.HandlerFunc {
	return func(context *gin.Context) {
		args := make([]reflect.Value, vm.Type().NumIn())
		mCopy := true
		// 【1】根据接口参数类型，封装处理参数+校验
		for i := 0; i < vm.Type().NumIn(); i++ {
			p := vm.Type().In(i)
			if strings.HasSuffix(p.String(), "gin.Context") {
				args[i] = reflect.ValueOf(context)
			} else if p == reflect.TypeOf(SecToken{}) {
				// 自动获取token对象，token由SecFilter提供
				v, e := context.Get("AccessToken")
				if e {
					args[i] = reflect.ValueOf(v)
				}
			} else if p.Kind() == reflect.Struct || (p.Kind() == reflect.Pointer && p.Elem().Kind() == reflect.Struct) {
				var data reflect.Value

				// 【1】初始化结构体
				if p.Kind() == reflect.Struct {
					// 【结构体】,直接new
					data = reflect.New(p)
				} else {
					// 【指针类型，并且是结构体的】,取得指针的【值类型】type，再做new操作
					data = reflect.New(p.Elem())
				}

				// 【2】参数结构化
				b := binding.Default(context.Request.Method, context.ContentType())
				if binding.JSON == b {
					// json格式，使用扩展json方式
					if err := fast_base.Json.NewDecoder(context.Request.Body).Decode(data.Interface()); err != nil {
						JSONIter(context, http.StatusOK, fast_base.Error(500, err.Error()))
						return
					}
				} else {
					// 使用gin自带的，用于处理form
					err := context.Bind(data.Interface())
					if err != nil {
						JSONIter(context, http.StatusOK, fast_base.Error(500, err.Error()))
						return
					}
				}

				// 【3】校验绑定后的参数
				if err := Validate.Struct(data.Interface()); err != nil {
					msg, _ := GetErrorStr(data.Interface(), err)
					JSONIter(context, http.StatusOK, fast_base.Error(500, msg))
					return
				}

				// 【4】 结果处理
				if p.Kind() == reflect.Struct {
					// new出来的是指针，需要转值类型
					args[i] = data.Elem()
				} else {
					// new出来的是指针，参数也是指针。所以直接设置
					args[i] = data
				}

			} else if p.Kind() == reflect.String {
				// go 反射无法获取参数名
				pv := context.Param(p.Name())
				args[i] = reflect.ValueOf(pv)
			} else if mCopy && p.Kind() == reflect.Map {
				// 创建目标 map 的实例
				targetMap := reflect.MakeMap(p)

				if mCopy {
					v := make(map[string]interface{})

					err := context.ShouldBindJSON(&v)
					if err != nil {
						JSONIter(context, http.StatusOK, fast_base.Error(500, err.Error()))
						return
					}
					// 将 tempMap 中的值放入 newMap
					for key, value := range v {
						targetMap.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
					}

					args[i] = targetMap

				} else {
					// 定义目标 map 类型，这里是 map[string]interface{}
					// 将请求体中的 JSON 绑定到 map
					v := targetMap.Interface()
					err := context.ShouldBindJSON(&v)
					if err != nil {
						JSONIter(context, http.StatusOK, fast_base.Error(500, err.Error()))
						return
					}
					args[i] = reflect.ValueOf(v)
				}

			}
		}

		//【2】调用接口
		re := vm.Call(args)

		//【3】结果处理
		if vm.Type().NumOut() == 0 {
			// 无返回值，直接标记成功
			JSONIter(context, http.StatusOK, fast_base.Success("成功"))
		} else {
			// 有返回值，识别数据和err
			v := re[vm.Type().NumOut()-1]              // 取最后一个返回值，如果有err则放最后一个
			r := vm.Type().Out(vm.Type().NumOut() - 1) // 取最后一个返回值类型
			if r.String() == "error" {
				if !v.IsNil() {
					err := v.Interface().(error)
					// 有错误信息，则抛返回错误
					JSONIter(context, http.StatusOK, fast_base.Error(500, err.Error()))
					return
				} else if vm.Type().NumOut()-2 < 0 {
					// 无err,并且前面也无返回值，则直接返回成功
					JSONIter(context, http.StatusOK, fast_base.Success("成功"))
					return
				} else {
					// 无err,但前面有返回值，则取出前面一个作为结果数据
					v = re[vm.Type().NumOut()-2]
					r = vm.Type().Out(vm.Type().NumOut() - 2)
				}
			}

			// 确定返回值后，封装成json
			if !v.IsValid() {
				JSONIter(context, http.StatusOK, fast_base.Success("成功"))
			} else {
				if reflect.TypeOf(fast_base.R{}) == r {
					// 如果返回就是r结构体，则直接返回
					JSONIter(context, http.StatusOK, v.Interface())
				} else {
					// 如果返回不是r结构体，则直接封装成R，保持统一返回结构
					JSONIter(context, http.StatusOK, fast_base.Success("成功").SetData(v.Interface()))
				}
			}
		}
	}
}

func (c *Server) Run() *Server {
	proxy.StartProxy()
	c.Gin.Run(ConfigServer.Address())
	return c
}

func (c *Server) RunAsService() *Server {
	c.HttpServer = &http.Server{
		Addr:    ConfigServer.Address(),
		Handler: c.Gin.Handler(),
	}
	go func() {
		proxy.StartProxy()
		fast_base.Logger.Info(fmt.Sprintf("Listening and serving HTTP on %s\n", ConfigServer.Address()))
		// service connections
		if err := c.HttpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	return c
}

func (c *Server) Shutdown() *Server {
	c.HttpServer.Shutdown(context.Background())
	proxy.StopProxy()
	return c
}
