package fast_utils

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/google/uuid"
	"strconv"
)

// 生成随机盐
func GenerateSalt(l int) (string, error) {
	salt := make([]byte, l)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	return hex.EncodeToString(salt), nil
}

// 加密密码，使用专属盐和固定值"123"
func HashPasswordWithSalt(password, salt string) string {
	str := "96" + salt + password + "69"
	//hashedPassword, err := bcrypt.GenerateFromPassword([]byte(combined), bcrypt.DefaultCost)
	//return string(hashedPassword), err
	data := []byte(str) //切片
	has := md5.Sum(data)
	md5str := fmt.Sprintf("%x", has) //将[]byte转成16进制
	return md5str
}

// 加密密码，使用专属盐和固定值"123"
func GetUUIDStr() string {
	return uuid.New().String()
}

func IntToStr(v int64) string {
	return strconv.FormatInt(v, 10)
}
