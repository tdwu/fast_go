package fast_base

import (
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"testing"
)

// test 测试函数
func TestDist(t *testing.T) {
	jStr := "{\n    \"id\": \"863857552633364480\",\n    \"type\": \"1001_1001_1001\",\n    \"typeName\": \"人身体伤亡_高处坠落事故_临边作业类\",\n    \"name\": \"安全测试\",\n    \"faceId\": \"863857543452033024\",\n    \"fileId\": \"1397757897536771933\",\n    \"size\": null,\n    \"md5\": \"1\",\n    \"status\": \"0\",\n    \"title\": \"编辑视频\",\n    \"gs\": \"1\",\n    \"lx\": \"1\",\n    \"note\": \"asdfafsdafasdfa\",\n    \"sc\": \"08:16\"\n}"
	b := TD{}
	// 设置json扩展器
	jsoniter.RegisterExtension(&JsonExtension{})
	error := jsoniter.Unmarshal([]byte(jStr), &b)
	if error != nil {
		fmt.Sprintf("%s", error.Error())
	}
	fmt.Sprintf("%s", b.Name)

}

type TD struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	FaceId string `json:"faceId"`
	FileId string `json:"fileId"`
	Size   int    `json:"size"`
	Status int    `json:"status"`
	Md5    string `json:"md5"`
	Lx     string `json:"lx"`   // 类型
	Sc     string `json:"sc"`   // 时长
	Gs     string `json:"gs"`   // 格式
	Note   string `json:"note"` // 描述
}
