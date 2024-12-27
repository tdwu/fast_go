package fast_base

import (
	"database/sql/driver"
	"errors"
	"strconv"
)

// 分页查询参数相关
type PageParam interface {
	GetIndex() int
	GetSize() int
}

type PageParams struct {
	PageIndex int `json:"pageIndex"`
	PageSize  int `json:"pageSize"`
}

func (that PageParams) GetIndex() int {
	if that.PageIndex <= 0 {
		return 1
	}
	return that.PageIndex
}

func (that PageParams) GetSize() int {
	if that.PageSize <= 0 {
		return 10
	}
	return that.PageSize
}

// 分页查询结果相关
type PageResult[T any] struct {
	PageParams
	TotalPages int  `json:"totalPages"`
	TotalRows  int  `json:"totalRows"`
	List       *[]T `json:"list"`
}

func (that *PageResult[T]) From(param PageParam) *PageResult[T] {
	that.PageIndex = param.GetIndex()
	that.PageSize = param.GetSize()
	return that
}

func (that *PageResult[T]) Set(count int64, list *[]T) {
	that.TotalRows = int(count)
	that.TotalPages = that.TotalRows / that.PageSize
	if that.TotalRows%that.PageSize > 0 {
		that.TotalPages = that.TotalPages + 1
	}
	that.List = list
}

// StringInt64 自定义类型，用于int64到string转换
// Deprecated  int64返回前端时，通过json序列化时解决。保留代码作为数据库到前端使用自定义类型的demo
type StringInt64 int64

func (that StringInt64) ToString() string {
	return strconv.FormatInt(int64(that), 10)
}
func (that StringInt64) ToInt64() int64 {
	return int64(that)
}

// Value 实现 driver.Valuer 接口，写入数据库时转换为 int64
func (i StringInt64) Value() (driver.Value, error) {
	return int64(i), nil
}

// Scan 实现 sql.Scanner 接口，从数据库读取时转换为 int64
func (i *StringInt64) Scan(value interface{}) error {
	if v, ok := value.(int64); ok {
		*i = StringInt64(v)
		return nil
	}
	return errors.New("failed to scan StringInt64")
}

// R 统一公共返回
type R struct {
	Code    int         `json:"code"`    // 状态码
	Message string      `json:"message"` // 提示信息
	Data    interface{} `json:"data"`    // 数据
}

func (r R) SetData(value interface{}) R {
	r.Data = value
	return r
}

func (r R) SetMessage(message string) R {
	r.Message = message
	return r
}
func (r R) SetCode(code int) R {
	r.Code = code
	return r
}

func Success(message string) R {
	return R{
		Code:    200,
		Message: message,
	}
}

func Error(code int, message string) R {
	return R{
		Code:    code,
		Message: message,
	}
}
