package fast_db

import (
	"fast_base"
	"fmt"
	"gorm.io/gorm"
	"strings"
)

// QueryPageListByDB 执行query
func QueryPageListByDB[T any](param fast_base.PageParam, query *gorm.DB) (*fast_base.PageResult[T], error) {

	// 创建 PageResult
	r := fast_base.PageResult[T]{}
	r.From(param)
	// 查询总数
	var count int64
	query.Count(&count)

	// 设置分页参数
	query = query.Limit(r.PageSize).Offset((r.PageIndex - 1) * r.PageSize)
	// 查询分页数据
	var results []T
	query.Find(&results)

	// 设置分页结果
	r.Set(count, &results)

	return &r, nil
}

// QueryPageListBySql 用于直接执行 SQL 查询，并将结果封装到指定的结构体中
func QueryPageListBySql[T any](param fast_base.PageParam, sql string, params ...interface{}) (*fast_base.PageResult[T], error) {
	// 创建 PageResult
	r := fast_base.PageResult[T]{}
	// 设置分页信息，假设从 params 中提取分页参数
	r.From(param)

	var results []T
	// 拼接 SQL 查询
	offset := (r.PageIndex - 1) * r.PageSize
	limitSQL := fmt.Sprintf("LIMIT %d, %d", offset, r.PageSize)

	// 执行查询
	db := DB
	err := db.Raw(sql+" "+limitSQL, params...).Scan(&results).Error
	if err != nil {
		return nil, err
	}

	// 查询总数
	var count int64
	countSQL := "SELECT COUNT(*) FROM (" + RemoveOrderBy(sql) + ") AS total_count"
	err = db.Raw(countSQL, params...).Scan(&count).Error
	if err != nil {
		return nil, err
	}

	// 设置分页结果
	r.Set(count, &results)

	return &r, nil
}

// GetListBySql 用于直接执行 SQL 查询，并将结果封装到指定的结构体中
func GetListBySql[T any](sql string, params ...interface{}) (*[]T, error) {
	var results []T
	// 执行查询
	db := DB
	err := db.Raw(sql, params...).Scan(&results).Error
	if err != nil {
		return nil, err
	}

	// 查询总数

	return &results, nil
}

func RemoveOrderBy(sql string) string {
	// 找到 ORDER BY 子句并去除
	orderByIndex := strings.Index(strings.ToUpper(sql), "ORDER BY")
	if orderByIndex != -1 {
		return sql[:orderByIndex]
	}
	return sql
}

func GetById[T any](id interface{}) *T {
	var result T
	if err := DB.First(&result, id).Error; err != nil {
		return nil // 发生错误
	}
	return &result
}

func GetOne[T any](sql string, params ...interface{}) *T {
	var result T
	if err := DB.Raw(sql, params...).First(&result).Error; err != nil {
		return nil // 发生错误
	}
	return &result
}

func CheckExists(sql string, params ...interface{}) bool {
	var count int64
	result := DB.Raw(sql, params...).Count(&count)
	if count > 0 {
		return true
	}
	if result.Error != nil {
		fast_base.Logger.Info("查询异常：" + result.Error.Error())
	}
	return false

}

func CountNum(sql string, params ...interface{}) int {
	var count int64
	DB.Raw(sql, params...).Count(&count)
	return int(count)

}
