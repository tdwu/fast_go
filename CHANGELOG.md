# Changelog

## Unreleased

### fast_base v0.7.0

- Go 版本提升至 1.26.5，升级 Viper、Zap 及其传递依赖。
- JSON 扩展改为框架专属 API，修复窄整数和浮点数反序列化时的越界写入风险。
- JSON 兼容解码加入范围检查，int64 输出继续以字符串表示。

### fast_web v0.7.0

- 新增无反射的 `JSONHandler` 与 `JSONHandlerWithToken` 泛型接口。
- 请求绑定、校验、令牌读取和响应序列化集中处理；新增集成测试。
- 旧反射路由保持兼容，函数签名仅在路由注册时解析一次。

### fast_db v0.7.0

- 升级到 `fast_base/v0.7.0`、Zap 1.28 和 GORM 1.31.2。

### fast_utils v0.7.0

- Go 版本提升至 1.26.5。

### fast_wgen v0.7.0

- 升级 Swag 的传递依赖；路由生成改为确定性、格式化和原子写入。
