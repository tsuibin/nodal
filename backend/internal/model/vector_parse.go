package model

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var errUnsupportedVectorSrc = errors.New("pgvector: 不支持的扫描源类型")

// appendFloat 以最短可逆形式追加一个 float32。
func appendFloat(buf []byte, f float32) []byte {
	return strconv.AppendFloat(buf, float64(f), 'g', -1, 32)
}

// parseVectorString 解析 pgvector 的文本格式 "[1,2,3]"。
func parseVectorString(s string, dst *Vector) error {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	if s == "" {
		*dst = nil
		return nil
	}

	parts := strings.Split(s, ",")
	vec := make(Vector, 0, len(parts))
	for _, p := range parts {
		f, err := strconv.ParseFloat(strings.TrimSpace(p), 32)
		if err != nil {
			return fmt.Errorf("pgvector: 解析分量 %q 失败: %w", p, err)
		}
		vec = append(vec, float32(f))
	}
	*dst = vec
	return nil
}
