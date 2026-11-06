package model

import (
	"database/sql/driver"
	"testing"
)

func TestVectorValue(t *testing.T) {
	v := Vector{0.1, 0.2, -0.3}
	val, err := v.Value()
	if err != nil {
		t.Fatalf("Value 失败: %v", err)
	}
	if val.(string) != "[0.1,0.2,-0.3]" {
		t.Fatalf("序列化不符: %s", val)
	}

	var empty Vector
	if got, _ := empty.Value(); got != nil {
		t.Fatalf("空向量应返回 nil，实际 %v", got)
	}
}

func TestVectorScan(t *testing.T) {
	var v Vector
	if err := v.Scan("[1,2,3]"); err != nil {
		t.Fatalf("Scan 失败: %v", err)
	}
	if len(v) != 3 || v[0] != 1 || v[2] != 3 {
		t.Fatalf("解析结果不符: %v", v)
	}

	if err := v.Scan([]byte("[0.5]")); err != nil || len(v) != 1 || v[0] != 0.5 {
		t.Fatalf("[]byte 扫描失败: %v %v", v, err)
	}

	if err := v.Scan(nil); err != nil || v != nil {
		t.Fatalf("nil 扫描应清空: %v %v", v, err)
	}

	if err := v.Scan("[1,abc]"); err == nil {
		t.Fatal("非法分量应报错")
	}

	if err := v.Scan(12345); err == nil {
		t.Fatal("不支持的类型应报错")
	}
}

// 确保 Vector 满足 driver.Valuer / sql.Scanner 接口
var (
	_ driver.Valuer                = Vector{}
	_ interface{ Scan(any) error } = (*Vector)(nil)
)
