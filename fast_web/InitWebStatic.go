package fast_web

import (
	"fast_web/web"
	"github.com/gin-gonic/gin"
	"net/http"
	"path"
	"reflect"
	"strings"
)

func (c *Server) LoadStatic(relativePath, root string) *Server {
	c.LoadStaticFs(relativePath, gin.Dir(root, false))
	return c
}
func (c *Server) LoadStaticFs(relativePath string, fs http.FileSystem) *Server {
	if strings.Contains(relativePath, ":") || strings.Contains(relativePath, "*") {
		panic("URL parameters can not be used when serving a static folder")
	}
	handler := createStaticHandler(relativePath, fs)
	urlPattern := path.Join(relativePath, "/*filepath")

	// Register GET and HEAD handlers
	c.Gin.GET(urlPattern, handler)
	//group.HEAD(urlPattern, handler)
	return c
}

func createStaticHandler(relativePath string, fs http.FileSystem) gin.HandlerFunc {
	//absolutePath := group.calculateAbsolutePath(relativePath)
	absolutePath := relativePath
	// 文件系统包装了一次,包装了原始的Dir，即：GinDir->Dir
	// 调用联调：HandlerFunc-->fileServer.ServeHTTP-->fs1.ServeHTTP->serveFile->serveContent
	fileServer := http.StripPrefix(absolutePath, web.FileServer(fs))
	return func(c *gin.Context) {
		v := reflect.ValueOf(fs)
		if strings.Contains(v.String(), "gin.onlyFilesFS") {
			// 只能查看文件的情况，先标记为404.读取到了后再修改
			c.Writer.WriteHeader(http.StatusNotFound)
		}
		/*
			// 不知道为什么要在这地方检查，由FileServer判断不行吗？所以先注释掉
			file := c.Param("filepath")
			// Check if file exists and/or if we have permission to access it
			f, err := fs.Open(file)
			if err != nil {
				c.Writer.WriteHeader(http.StatusNotFound)
				c.handlers = group.engine.noRoute
				// Reset index
				c.index = -1
				return
			}
			f.Close()
		*/

		fileServer.ServeHTTP(c.Writer, c.Request)
	}
}
