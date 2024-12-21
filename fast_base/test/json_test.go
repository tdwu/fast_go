package test

import (
	"bytes"
	"fmt"
	"github.com/bytedance/sonic"
	"testing"
)

func TestJson(t *testing.T) {
	p := SonicPerson{
		Name:   "张三",
		IsMale: true,
		Score:  100,
	}

	// 使用Sonic进行序列化
	var buf bytes.Buffer
	enc := sonic.ConfigDefault.NewEncoder(&buf)
	if err := enc.Encode(p); err != nil {
		fmt.Println("序列化出错:", err)
		return
	}

	// 输出序列化后的JSON字符串
	fmt.Println(buf.String())
}

// 定义一个数据字典，用于转换字段值
var valueMap = map[string]string{
	"true":  "yes",
	"false": "no",
	"100":   "满分",
}

type SonicPerson struct {
	Name   string `json:"name"`
	IsMale bool   `json:"is_male"` // 使用string类型以便转换
	Score  int    `json:"score"`
}

func (p *SonicPerson) Marshal(enc *sonic.Encoder) error {

	//	enc.
	// 序列化Name字段
	if err := (*enc).Encode(p.Name); err != nil {
		return err
	}
	// 根据数据字典转换IsMale字段的值
	if p.IsMale {
		if err := (*enc).Encode("yes"); err != nil {
			return err
		}
	} else {
		if err := (*enc).Encode("no"); err != nil {
			return err
		}
	}

	// 序列化Score字段
	if err := (*enc).Encode(p.Score); err != nil {
		return err
	}

	return nil
}
