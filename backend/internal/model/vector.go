package model

import "database/sql/driver"

// Vector 是 pgvector 的 Go 表示。
// 不引入 pgvector-go 依赖：Value 输出 "[0.1,0.2,...]" 字符串字面量，Scan 解析回来，
// PostgreSQL 的 vector 类型可直接接受该文本格式。
type Vector []float32

// Value 实现 driver.Valuer。
func (v Vector) Value() (driver.Value, error) {
	if len(v) == 0 {
		return nil, nil
	}
	buf := make([]byte, 0, len(v)*10+2)
	buf = append(buf, '[')
	for i, f := range v {
		if i > 0 {
			buf = append(buf, ',')
		}
		buf = appendFloat(buf, f)
	}
	buf = append(buf, ']')
	return string(buf), nil
}

// Scan 实现 sql.Scanner。
func (v *Vector) Scan(src any) error {
	switch s := src.(type) {
	case nil:
		*v = nil
		return nil
	case string:
		return parseVectorString(s, v)
	case []byte:
		return parseVectorString(string(s), v)
	default:
		return errUnsupportedVectorSrc
	}
}

// Dimensions 返回向量维度（0 表示空）。
func (v Vector) Dimensions() int { return len(v) }
