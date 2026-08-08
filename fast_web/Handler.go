package fast_web

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tdwu/fast_go/fast_base"
)

// Handler 是推荐的新接口形态：请求和响应类型在编译期确定，避免旧路由包装器
// 在每个请求中扫描函数签名和拼装 reflect.Value。
//
// 例：
//
//	gin.POST("/users", fast_web.JSONHandler(CreateUser))
//	func CreateUser(c *gin.Context, req *CreateUserRequest) (User, error) { ... }
type Handler[Request any, Response any] func(*gin.Context, *Request) (Response, error)

// JSONHandler 将 JSON/form/URI 参数绑定、校验、统一响应集中在一处。请求结构体
// 应通过 json/form/uri 标签声明字段，避免把未命名的基础类型作为处理函数参数。
func JSONHandler[Request any, Response any](handler Handler[Request, Response]) gin.HandlerFunc {
	return func(c *gin.Context) {
		request, err := Bind[Request](c)
		if err != nil {
			writeRequestError(c, err)
			return
		}

		response, err := handler(c, request)
		if err != nil {
			writeApplicationError(c, err)
			return
		}
		writeResponse(c, response)
	}
}

// TokenHandler 是需要认证令牌的 JSONHandler 版本。它不依赖反射，并会在令牌
// 缺失或类型不匹配时返回 401，而不是让 reflect.Call 在运行时 panic。
type TokenHandler[Request any, Response any] func(*gin.Context, *Request, SecToken) (Response, error)

func JSONHandlerWithToken[Request any, Response any](handler TokenHandler[Request, Response]) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := AccessToken(c)
		if !ok {
			c.Abort()
			JSONIter(c, http.StatusUnauthorized, fast_base.Error(http.StatusUnauthorized, "未认证或登录已过期"))
			return
		}

		request, err := Bind[Request](c)
		if err != nil {
			writeRequestError(c, err)
			return
		}

		response, err := handler(c, request, token)
		if err != nil {
			writeApplicationError(c, err)
			return
		}
		writeResponse(c, response)
	}
}

// Bind 填充并校验请求对象。JSON 使用 fast_base.Json，保证与响应端一致的
// int64 序列化和历史数值兼容规则；form、query 和 uri 交给 Gin 的 binder。
func Bind[Request any](c *gin.Context) (*Request, error) {
	request := new(Request)
	if err := c.ShouldBindUri(request); err != nil {
		return nil, err
	}
	if err := c.ShouldBindQuery(request); err != nil {
		return nil, err
	}
	contentType := c.ContentType()
	if strings.Contains(contentType, "json") {
		if err := fast_base.Json.NewDecoder(c.Request.Body).Decode(request); err != nil {
			return nil, err
		}
	} else if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		if err := c.ShouldBind(request); err != nil {
			return nil, err
		}
	}

	LoadValidator()
	if err := Validate.Struct(request); err != nil {
		if message, ok := GetErrorStr(request, err); ok && message != "" {
			return nil, errors.New(message)
		}
		return nil, err
	}
	return request, nil
}

// AccessToken 提供受认证处理器的显式依赖，不再由反射层隐式注入。
func AccessToken(c *gin.Context) (SecToken, bool) {
	value, ok := c.Get("AccessToken")
	if !ok {
		return SecToken{}, false
	}
	token, ok := value.(SecToken)
	return token, ok
}

func writeRequestError(c *gin.Context, err error) {
	c.Abort()
	JSONIter(c, http.StatusBadRequest, fast_base.Error(http.StatusBadRequest, err.Error()))
}

func writeApplicationError(c *gin.Context, err error) {
	c.Abort()
	JSONIter(c, http.StatusInternalServerError, fast_base.Error(http.StatusInternalServerError, err.Error()))
}

func writeResponse[Response any](c *gin.Context, response Response) {
	// 业务层可显式返回 R，其他返回值会包裹为统一的成功响应。
	switch value := any(response).(type) {
	case fast_base.R:
		JSONIter(c, http.StatusOK, value)
	case *fast_base.R:
		if value == nil {
			JSONIter(c, http.StatusOK, fast_base.Success("成功"))
			return
		}
		JSONIter(c, http.StatusOK, *value)
	default:
		JSONIter(c, http.StatusOK, fast_base.Success("成功").SetData(response))
	}
}
