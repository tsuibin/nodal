package migrate

import (
	"testing"

	"github.com/nodal/nodal/backend/internal/model"
)

// tablenames 把模型列表映射成表名集合。
// 不用 gorm.Statement 解析（需要已初始化的 Dialector），这里直接按类型取 TableName()。
func tablenames(t *testing.T, models []any) map[string]bool {
	t.Helper()
	set := make(map[string]bool, len(models))
	for _, m := range models {
		switch v := m.(type) {
		case *model.User:
			set[v.TableName()] = true
		case *model.Canvas:
			set[v.TableName()] = true
		case *model.AgentTemplate:
			set[v.TableName()] = true
		case *model.CanvasNode:
			set[v.TableName()] = true
		case *model.CanvasEdge:
			set[v.TableName()] = true
		case *model.KnowledgeFile:
			set[v.TableName()] = true
		case *model.KnowledgeChunk:
			set[v.TableName()] = true
		case *model.CanvasLike:
			set[v.TableName()] = true
		case *model.CanvasComment:
			set[v.TableName()] = true
		default:
			t.Fatalf("未登记表名的模型类型: %T", m)
		}
	}
	return set
}

// 编译期断言：确保 model.Vector 满足维度约定
var _ = model.Vector(nil).Dimensions()
