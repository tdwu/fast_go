# 架构与发布约定

## 模块边界

本仓库是一个 Go workspace，不是一个单一发布包。`fast_base`、`fast_utils`、`fast_db`、`fast_web` 和 `fast_wgen` 都有独立的 `go.mod` 与导入路径；`go.work` 只用于本地联调，不能作为消费者依赖或发布版本。

依赖方向应保持为：`fast_utils` / `fast_base` 在底层，`fast_db` 与 `fast_web` 依赖 `fast_base`，`fast_web` 还依赖 `fast_utils`，`fast_wgen` 是开发期代码生成工具。底层模块绝不能导入上层模块。

## Web 接口约定

新接口使用类型化处理器，避免旧版 `GenHandlerFunc(reflect.ValueOf(...))` 在运行时填充参数和调用函数：

```go
type CreateUserRequest struct {
    Name string `json:"name" validate:"required"`
}

type User struct { ID int64 `json:"id"` }

func CreateUser(c *gin.Context, req *CreateUserRequest) (User, error) {
    return User{ID: 9007199254740993}, nil
}

router.POST("/users", fast_web.JSONHandler(CreateUser))
```

`JSONHandler` 统一执行 URI/query/body 绑定、`validate` 校验、错误响应和 JSON 序列化；`JSONHandlerWithToken` 显式注入 `SecToken`。请求字段应使用 `json`、`form`、`uri` 标签，避免未命名的基础类型参数。业务函数返回 `fast_base.R` 时保持原响应，否则自动包装成统一的 `R`。

旧的反射路由兼容层仍可运行，但只适合渐进迁移；它不会再在每次请求中重复解析函数签名。

接口说明继续以源码注释为单一事实来源：路由用 `@router /path [post]`，OpenAPI 使用 Swag 的 `@Summary`、`@Param`、`@Success` 等注释。`fast_wgen` 通过 Go AST 分析这些注释，生成器本身不依赖运行时反射，并以原子写入生成格式化的路由文件。

## 发布与版本管理

只修改一个模块时，通常只发布那个模块：

| 变更 | 必须发布 | 何时连带发布 |
| --- | --- | --- |
| `fast_utils` | `fast_utils` | 上层模块将最低依赖升级到新 API 时 |
| `fast_base` | `fast_base` | `fast_db` / `fast_web` 使用了新 API 或需锁定新最小版本时 |
| `fast_db` | `fast_db` | 无 |
| `fast_web` | `fast_web` | 无 |
| `fast_wgen` | `fast_wgen` | 仅使用生成器的项目需要升级时 |

发布前在每个要发布模块目录运行 `GOWORK=off go test ./...`，这样可验证外部消费者的真实模块解析；再运行全工作区 `go test ./fast_base/... ./fast_db/... ./fast_utils/... ./fast_web/... ./fast_wgen/...` 验证集成。不要把本地 `go.work` 的 replace 效果当成发布成功。

Git tag 必须使用子模块前缀，例如 `fast_web/v0.7.0`、`fast_base/v0.7.0`。补丁修复用 PATCH，新功能且兼容用 MINOR；删除导出 API、改变 JSON 字段/HTTP 语义或升级到 `v2` 路径时才用 MAJOR。当前最新 tag 为 `v0.6.0`，建议本次兼容性改造发布为 `fast_base/v0.7.0`、`fast_web/v0.7.0`，并在 `CHANGELOG.md` 记录迁移期和废弃项。

发布顺序始终从依赖图底部向上：先 `fast_utils/v0.6.1` 和 `fast_base/v0.7.0`，再 `fast_db/v0.6.1` / `fast_web/v0.7.0`，最后 `fast_wgen/v0.6.1`。本次上层模块已经要求尚未发布的版本，因此必须先合并并打底层 tag；在第一步完成前，`GOWORK=off` 验证上层模块会按预期找不到该版本。未使用新 API 的上层模块无需为了“统一版本号”而重新发布。
