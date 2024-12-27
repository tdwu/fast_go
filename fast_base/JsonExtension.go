package fast_base

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/modern-go/reflect2"
	"reflect"
	"strconv"
	"strings"
	"unsafe"
)

/**
使用jsoniter增强json序列化和反序列化的能力
1 序列化时根据数据字典，自动将编码value转成字面值用于前端显示。自动新增字段名称，不影响原有值
2 序列化int64时，自动转成string。解决前端js精度问题。
3 序列化时，根据id关联出从表的字段。
4 反序列化时，处理带引号的数值类型（严格说string）无法转换成数值问题问题。如"1"无法转换成int。

*/
// 创建配置
var json = jsoniter.ConfigCompatibleWithStandardLibrary

// DictCodec 1 数据字典转换
type DictCodec struct {
	originalEncoder jsoniter.ValEncoder
	filedName       string
	dictName        string
	getValue        GetOriginalValue
}

type GetOriginalValue func(ptr unsafe.Pointer) string

func (d *DictCodec) IsEmpty(ptr unsafe.Pointer) bool {
	// 一致不为空
	return false
}

// Encode 序列化时的增强代码（数据字典）
func (d *DictCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {

	if d.filedName != "" {
		d.originalEncoder.Encode(ptr, stream) // 序列化原始值
		stream.WriteMore()                    // 添加逗号
		stream.WriteObjectField(d.filedName)  // 输出补充的字段名
	}

	dictMap := DictCenter[d.dictName]
	if dictMap != nil {
		if name, found := dictMap[d.getValue(ptr)]; found {
			stream.WriteString(name) // 输出映射值
		} else {
			stream.WriteString("")
		}
	} else {

		stream.WriteString("")
	}
}

// DictSqlCodec 2 sql增强
type DictSqlCodec struct {
	originalEncoder jsoniter.ValEncoder
	filedName       string
	sql             string
	getValue        GetOriginalValue
}

// QueryBySql 2 SQL查询转换
type QueryBySql func(sql string, p ...interface{}) string

// DictQueryBySql 注：由DB模块去实现, 包括缓存
var DictQueryBySql QueryBySql

func (d *DictSqlCodec) IsEmpty(ptr unsafe.Pointer) bool {
	// 一致不为空
	return false
}

// Encode 序列化时的增强代码（SQL）
func (d *DictSqlCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {

	d.originalEncoder.Encode(ptr, stream) // 序列化原始 Sex 值
	if d.filedName == "" {
		return
	}

	stream.WriteMore()                   // 添加逗号
	stream.WriteObjectField(d.filedName) // 输出补充的字段名

	if DictQueryBySql == nil {
		// 没有实现的情况
		stream.WriteString("")
		return
	}

	newValue := DictQueryBySql(d.sql, d.getValue(ptr))
	stream.WriteString(newValue)

}

// JsonExtension 3 扩展器
type JsonExtension struct {
	jsoniter.DummyExtension
}

// CreateEncoder 序列化
func (ext *JsonExtension) CreateEncoder(typ reflect2.Type) jsoniter.ValEncoder {
	if typ.Kind() == reflect.Int64 {
		return &GlobalWrapCodec{
			encodeFunc: func(ptr unsafe.Pointer, stream *jsoniter.Stream) {
				//  int64，前端js无法支持。所以转换成字符串
				val := *(*int64)(ptr)
				stream.WriteString(strconv.FormatInt(val, 10))
			},
			isEmptyFunc: nil}
	}
	return nil
}

// CreateDecoder 反序列化
func (ext *JsonExtension) CreateDecoder(typ reflect2.Type) jsoniter.ValDecoder {
	switch typ.Kind() {
	case reflect.String: // 目标类型，如果是字符串。输入数字也能转换成功
		return &GlobalWrapCodec{
			decodeFunc: func(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
				switch iter.WhatIsNext() {
				case jsoniter.NumberValue:
					// 如果是数字类型，转换为字符串
					number := iter.ReadNumber()
					str := number.String()
					*(*string)(ptr) = str
				case jsoniter.StringValue:
					// 如果是字符串，直接读取
					*(*string)(ptr) = iter.ReadString()
				default:
					iter.Read()
				}
			},
		}
	case reflect.Int, reflect.Int32, reflect.Int64: // 目标是int。如果是字符串，也能转换。
		return &GlobalWrapCodec{
			decodeFunc: func(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
				switch iter.WhatIsNext() {
				case jsoniter.StringValue: // 目标是int。如果是字符串，也能转换。
					// 读取字符串并转换为 int64
					strVal := iter.ReadString()

					if strVal == "" {
						return
					}

					intVal, err := strconv.ParseInt(strVal, 10, 64)
					if err != nil {
						iter.ReportError("NumericCompatibleDecoder", "invalid int format")
						return
					}
					*(*int64)(ptr) = intVal
				case jsoniter.NumberValue:
					// 直接读取数字并存储为 int64
					*(*int64)(ptr) = iter.ReadInt64()
				default:
					iter.Read()

				}
			},
		}
	case reflect.Float32, reflect.Float64: // 目标是小数。如果是字符串，也能转换。
		return &GlobalWrapCodec{
			decodeFunc: func(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
				switch iter.WhatIsNext() {
				case jsoniter.StringValue:
					// 读取字符串并转换为 float64
					strVal := iter.ReadString()
					floatVal, err := strconv.ParseFloat(strVal, 64)
					if strVal == "" {
						return
					}

					if err != nil {
						iter.ReportError("NumericCompatibleDecoder", "invalid float format")
						return
					}
					*(*float64)(ptr) = floatVal
				case jsoniter.NumberValue:
					// 直接读取数字并存储为 float64
					*(*float64)(ptr) = iter.ReadFloat64()
				default:
					var v float64
					*(*float64)(ptr) = v

					// 忽略掉
					//	iter.ReportError("NumericCompatibleDecoder", "unexpected value type")
				}
			},
		}

	}

	return nil
}

// UpdateStructDescriptor 根据注解中的描述符，启用对应的Codec
func (ext *JsonExtension) UpdateStructDescriptor(structDescriptor *jsoniter.StructDescriptor) {
	for _, binding := range structDescriptor.Fields {
		// 查找含有 "jsonDict" 标签的字段
		if dictTag := binding.Field.Tag().Get("jsonDict"); dictTag != "" {

			jsonName := binding.Field.Tag().Get("json")
			if jsonName == "-" {
				//忽略
				return
			}

			if jsonName == "" {
				// 没有注解，默认为字段名称
				jsonName = binding.Field.Name()
			}

			if strings.HasSuffix(jsonName, "Name") {
				jsonName = "" // 不新增字段
			} else {
				jsonName = jsonName + "Name" // 如果 json 标签为空，使用字段本身的名称
			}

			dc := &DictCodec{
				getValue:        getValueMethod(binding),
				originalEncoder: binding.Encoder,
				filedName:       jsonName,
				dictName:        dictTag,
			}
			// 为该字段添加自定义序列化逻辑
			binding.Encoder = dc
		}

		if sqlTag := binding.Field.Tag().Get("jsonSql"); sqlTag != "" {
			jsonName := binding.Field.Tag().Get("json")
			if jsonName == "-" {
				//忽略
				return
			}

			jn := binding.Field.Tag().Get("jsonName")
			if jn != "" {
				jsonName = jn
			} else {
				if jsonName == "" {
					// 没有注解，默认为字段名称
					jsonName = binding.Field.Name()
				}
				if strings.HasSuffix(jsonName, "Name") {
					jsonName = "" // 不新增字段
				} else if strings.HasSuffix(jsonName, "Id") {
					t := []byte(jsonName)
					jsonName = string(t[:len(t)-2]) + "Name" // ID换成Name
				} else {
					jsonName = jsonName + "Name" // 如果 json 标签为空，使用字段本身的名称
				}
			}

			dc := &DictSqlCodec{
				getValue:        getValueMethod(binding),
				originalEncoder: binding.Encoder,
				sql:             sqlTag,
				filedName:       jsonName,
			}
			// 为该字段添加自定义序列化逻辑
			binding.Encoder = dc
		}
	}
}

func getValueMethod(binding *jsoniter.Binding) GetOriginalValue {
	fieldType := binding.Field.Type().Type1()

	switch fieldType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return func(ptr unsafe.Pointer) string {
			return strconv.Itoa(*(*int)(ptr))
		}
	case reflect.String:
		return func(ptr unsafe.Pointer) string {
			return *(*string)(ptr)
		}
	default:
		return func(ptr unsafe.Pointer) string {
			return *(*string)(ptr)
		}
	}
	return func(ptr unsafe.Pointer) string {
		return *(*string)(ptr)
	}
}

// GlobalWrapCodec 通用wrap
type GlobalWrapCodec struct {
	encodeFunc  func(ptr unsafe.Pointer, stream *jsoniter.Stream)
	isEmptyFunc func(ptr unsafe.Pointer) bool
	decodeFunc  func(ptr unsafe.Pointer, iter *jsoniter.Iterator)
}

func (codec *GlobalWrapCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	codec.encodeFunc(ptr, stream)
}

func (codec *GlobalWrapCodec) IsEmpty(ptr unsafe.Pointer) bool {
	if codec.isEmptyFunc == nil {
		return false
	}
	return codec.isEmptyFunc(ptr)
}

func (codec *GlobalWrapCodec) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	codec.decodeFunc(ptr, iter)
}

/*

// MarshalJSON 实现 MarshalJSON 方法，将 int64 转换为字符串(废弃了)
func (i StringInt64) MarshalJSON() ([]byte, error) {
	return []byte(`"` + strconv.FormatInt(int64(i), 10) + `"`), nil
}

// UnmarshalJSON ：JSON解码，将字符串（带或不带引号）转回 int64
func (i *StringInt64) UnmarshalJSON(data []byte) error {
	strData := string(data)

	// 检查是否包含引号，如果有则去除
	if strData[0] == '"' && strData[len(strData)-1] == '"' {
		strData = strData[1 : len(strData)-1]
	}

	// 将字符串解析为 int64
	parsed, err := strconv.ParseInt(strData, 10, 64)
	if err != nil {
		return err
	}

	*i = StringInt64(parsed)
	return nil
}
*/
