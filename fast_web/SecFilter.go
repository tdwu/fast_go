package fast_web

import (
	"github.com/gin-gonic/gin"
	"github.com/tdwu/fast_go/fast_base"
	"golang.org/x/time/rate"
	"net/http"
	"strings"
)

// LoadLimit 限流
func (c *Server) LoadLimit() *Server {
	// 限流中间件，公共限流器，对所有接口都有效。
	Container.Gin.Use(RateLimitMiddleware(100, 200))
	return c
}

// LoadLimitByPassword 简单密码模式
func (c *Server) LoadLimitByPassword(prefix ...string) *Server {
	// 对API进行密码验证，适用于简单场景
	Container.Gin.Use(func(context *gin.Context) {
		if matchPrefix(context.Request.URL.Path, prefix) {
			ptt := context.Query("tt")
			ctt := fast_base.ConfigAll.GetString("server.password")
			if ctt == ptt {
				context.Next()
			} else {
				JSONIter(context, http.StatusOK, fast_base.Error(403, "请登录"))
				context.Abort()
			}
		} else {
			context.Next()
		}

	})
	return c
}

// LoadLimitByToken token模式
func (c *Server) LoadLimitByToken(prefix ...string) *Server {
	SecTokenController.Init()
	// 对API进行密码验证，适用于简单场景
	Container.Gin.Use(func(context *gin.Context) {
		/*if context.Request.URL.Path == "/api/sec/user/refreshToken" {
			// 刷新token
			refreshTokenCode := context.GetHeader("RefreshToken")
			accessTokenCode := context.GetHeader("AccessToken")
			accessToken := SecTokenController.GetAccessToken(accessTokenCode)
			refreshToken := SecTokenController.GetAccessToken(refreshTokenCode)
			if accessToken == nil || !accessToken.IsValid() || refreshToken == nil || !refreshToken.IsValid() {
				JSONIter(context,http.StatusOK, fast_base.Error(403, "token刷新失败"))
				context.Abort()
			}
			newToken := SecTokenController.RefreshNewToken(*refreshToken, refreshToken.Data)
			JSONIter(context,http.StatusOK, fast_base.Success("更新成功", newToken))
		}*/
		if matchPrefix(context.Request.URL.Path, prefix) {
			accessTokenCode := context.GetHeader("AccessToken")
			AppKey := context.GetHeader("AppKey")
			if accessTokenCode == "" {
				// 没有提供token
				JSONIter(context, http.StatusOK, fast_base.Error(401, "请登录"))
				context.Abort()
				return
			}
			accessToken := SecTokenController.GetAccessToken(AppKey, accessTokenCode)
			if accessToken == nil {
				// 根据code没获取到token
				JSONIter(context, http.StatusOK, fast_base.Error(402, "请重新登录"))
				context.Abort()
			} else {
				context.Set("AccessToken", *accessToken)
				context.Next()
			}
		} else {
			context.Next()
		}
	})

	return c
}

// RateLimitMiddleware
// num 每秒钟Token Bucket中会产生多少token
// cap 最多存在多少个可用的token。
// 不是所有接口都需要限流，建议对关键接口进行限流。
func RateLimitMiddleware(num int, cap int) gin.HandlerFunc {
	// 如果超过1秒的,如5秒一个，建议使用web/RateLimit
	//https://github.com/chenyahui/AnnotatedCode/blob/master/go/x/time/rate/rate.go
	limit := rate.NewLimiter(rate.Limit(num), cap)
	return func(c *gin.Context) {
		fast_base.Logger.Info("[Limit]：" + c.Request.URL.String())
		if !limit.Allow() {
			c.JSON(http.StatusOK, fast_base.Error(403, "无服务器繁忙，轻稍后再试"))
			c.Abort()
			return
		}
		c.Next()
	}
}

func matchPrefix(url string, prefix []string) bool {
	for _, s := range prefix {
		if strings.HasPrefix(url, s) {
			return true
		}
	}
	return false
}

// Access-Control-Allow-Origin: 这个头部指定了哪些域可以访问资源。这里设置为 * 表示允许所有域的访问。
// Access-Control-Allow-Methods: 指定了允许的 HTTP 方法（如 GET, POST, DELETE 等）。
// Access-Control-Allow-Headers: 指定了在跨域请求中可以使用的 HTTP 头部列表。
// Access-Control-Allow-Credentials: 表示是否允许发送 cookies。
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
