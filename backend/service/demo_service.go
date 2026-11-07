package service

// 模块 10：新用户 Demo 画布。
//
// Demo 不写死在数据库迁移里，而是在「该身份第一次拥有 0 张画布」时按需生成：
//   - 注册成功：AuthService.Register 直接种两份；
//   - 老账号/游客首次打开资源库：前端调用 POST /api/demo/seed 兜底。
//
// 生成逻辑复用 CanvasService.Create，因此节点/边同样走 normalizeDocument
// 校验，不会出现 Demo 绕过正式数据约束的情况。

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"gorm.io/gorm"

	"github.com/nodal/nodal-server/model"
	"github.com/nodal/nodal-server/token"
)

type DemoService struct {
	db       *gorm.DB
	canvases *CanvasService
}

func NewDemoService(db *gorm.DB) *DemoService {
	return &DemoService{db: db, canvases: NewCanvasService(db)}
}

// SeedResult 是 POST /api/demo/seed 的返回体。
type SeedResult struct {
	Created int      `json:"created"`
	Titles  []string `json:"titles"`
}

// Seed 在 owner 还没有任何画布时创建两张 Demo；已有画布则直接返回 0。
// 这样既保证新用户首次打开就有内容，也不会反复把用户删掉的画布补回来。
func (s *DemoService) Seed(ctx context.Context, id *token.Identity) (*SeedResult, error) {
	if id == nil || id.OwnerKey() == "" {
		return nil, fmt.Errorf("缺少身份")
	}

	var count int64
	if err := s.db.WithContext(ctx).Model(&model.Canvas{}).
		Where("owner_key = ?", id.OwnerKey()).Count(&count).Error; err != nil {
		return nil, fmt.Errorf("查询画布数量失败: %w", err)
	}
	if count > 0 {
		return &SeedResult{Created: 0, Titles: nil}, nil
	}

	result := &SeedResult{}
	for _, demo := range demoCanvases() {
		nodes, err := json.Marshal(demo.Nodes)
		if err != nil {
			return result, fmt.Errorf("序列化 Demo 节点失败: %w", err)
		}
		edges, err := json.Marshal(demo.Edges)
		if err != nil {
			return result, fmt.Errorf("序列化 Demo 连线失败: %w", err)
		}
		viewport, err := json.Marshal(demo.Viewport)
		if err != nil {
			return result, fmt.Errorf("序列化 Demo 视口失败: %w", err)
		}

		created, err := s.canvases.Create(ctx, id, CreateCanvasInput{
			Title:       demo.Title,
			Description: demo.Description,
			Nodes:       nodes,
			Edges:       edges,
			Viewport:    viewport,
			Tags:        demo.Tags,
		})
		if err != nil {
			return result, fmt.Errorf("创建 Demo 画布 %q 失败: %w", demo.Title, err)
		}
		result.Created++
		result.Titles = append(result.Titles, created.Title)
		slog.Info("demo canvas seeded", "canvas_id", created.ID, "owner_key", id.OwnerKey())
	}
	return result, nil
}

// SeedOnRegister 给刚注册的用户种 Demo；失败只记日志，不影响注册成功。
func (s *DemoService) SeedOnRegister(ctx context.Context, userID uint) {
	if userID == 0 {
		return
	}
	if _, err := s.Seed(ctx, &token.Identity{Role: token.RoleUser, UserID: userID}); err != nil {
		slog.Warn("seed demo canvases after register failed", "user_id", userID, "error", err)
	}
}

type demoCanvas struct {
	Title       string
	Description string
	Tags        []string
	Viewport    map[string]float64
	Nodes       []map[string]any
	Edges       []map[string]any
}

func demoNode(id string, x, y float64, data map[string]any) map[string]any {
	return map[string]any{
		"id":       id,
		"type":     "nodal",
		"position": map[string]float64{"x": x, "y": y},
		"data":     data,
	}
}

func demoEdge(id, source, target, relation string) map[string]any {
	return map[string]any{
		"id":     id,
		"source": source,
		"target": target,
		"type":   "relation",
		"data":   map[string]any{"relation": relation},
	}
}

func demoCanvases() []demoCanvas {
	return []demoCanvas{
		demoDebate(),
		demoCodeCompare(),
	}
}

// Demo1：辩论推演。展示一个命题如何分出正反方、再汇入结论。
func demoDebate() demoCanvas {
	nodes := []map[string]any{
		demoNode("demo-claim", 80, 150, map[string]any{
			"kind":    "text",
			"label":   "核心命题",
			"content": "## AI 生成内容是否应享有著作权？\n\n请沿着正反两条路径继续推演，最后落到一个可执行的权利归属方案。",
			"source":  "user",
			"status":  "normal",
		}),
		demoNode("demo-pro", 420, 40, map[string]any{
			"kind":    "text",
			"label":   "支持方：独创性来自人类编排",
			"content": "提示词设计、素材选择与多轮筛选体现了人类的创造性投入，AI 只是工具。",
			"source":  "user",
			"status":  "normal",
		}),
		demoNode("demo-con", 420, 270, map[string]any{
			"kind":    "text",
			"label":   "反例质疑：没有可识别的作者",
			"content": "生成结果具有随机性，难以证明某一人类对最终表达享有排他性权利。",
			"source":  "user",
			"status":  "normal",
		}),
		demoNode("demo-evidence", 760, 40, map[string]any{
			"kind":         "text",
			"label":        "佐证：人类实质性贡献标准",
			"content":      "美国版权局 2023 年指南要求作品包含可识别的「人类作者性」；纯 AI 输出不受版权保护。",
			"source":       "agent",
			"status":       "normal",
			"ai_generated": true,
			"agent_name":   "辩论研究员",
		}),
		demoNode("demo-hypo", 760, 270, map[string]any{
			"kind":         "text",
			"label":        "并行假设：分层权利模型",
			"content":      "把权利拆成「工具提供者 / 提示词作者 / 最终编排者」三层，按贡献比例分配收益。",
			"source":       "agent",
			"status":       "normal",
			"ai_generated": true,
			"agent_name":   "辩论研究员",
		}),
		demoNode("demo-conclusion", 1080, 150, map[string]any{
			"kind":    "text",
			"label":   "结论：以人类实质性贡献为锚点",
			"content": "## 结论\n\n只有当人类对最终表达作出可识别的实质性贡献时，才给予著作权；否则进入公共领域，并用标注与平台规则治理。",
			"source":  "user",
			"status":  "conclusion",
		}),
	}
	edges := []map[string]any{
		demoEdge("demo-debate-edge-1", "demo-claim", "demo-pro", "hypothesis"),
		demoEdge("demo-debate-edge-2", "demo-claim", "demo-con", "counter"),
		demoEdge("demo-debate-edge-3", "demo-pro", "demo-evidence", "evidence"),
		demoEdge("demo-debate-edge-4", "demo-con", "demo-hypo", "hypothesis"),
		demoEdge("demo-debate-edge-5", "demo-evidence", "demo-conclusion", "derive"),
		demoEdge("demo-debate-edge-6", "demo-hypo", "demo-conclusion", "derive"),
	}
	return demoCanvas{
		Title:       "Demo｜AI 著作权辩论推演",
		Description: "从核心命题分出正反方，补充证据与并行假设，最后收敛到「人类实质性贡献」标准。可直接 Fork 后换成你的议题。",
		Tags:        []string{"示例", "辩论", "AI 伦理"},
		Viewport:    map[string]float64{"x": 40, "y": 60, "zoom": 0.72},
		Nodes:       nodes,
		Edges:       edges,
	}
}

// Demo2：代码多方案对比。展示代码节点、方案证据、反例与组合结论。
func demoCodeCompare() demoCanvas {
	bloom := `from math import log

def bloom_size(n: int, p: float = 0.01) -> tuple[int, int]:
    """按预期元素数 n 与误判率 p 计算位图大小和哈希次数。"""
    m = -n * log(p) / (log(2) ** 2)
    k = (m / n) * log(2)
    return int(m), max(1, round(k))

print(bloom_size(1_000_000))`

	nullCache := `from cachetools import TTLCache

# 空值缓存：命中不存在的 key 时写一个短 TTL 的占位，
# 防止同一个随机 key 反复击穿数据库。
null_cache = TTLCache(maxsize=100_000, ttl=30)

def get_user(uid: str):
    if uid in null_cache:
        return None
    user = db.query(uid)
    if user is None:
        null_cache[uid] = True
    return user`

	nodes := []map[string]any{
		demoNode("demo-code-claim", 80, 160, map[string]any{
			"kind":    "text",
			"label":   "问题：缓存穿透怎么防？",
			"content": "高并发下大量不存在的 key 绕过缓存直查数据库。对比「布隆过滤器」与「空值缓存」两种方案。",
			"source":  "user",
			"status":  "normal",
		}),
		demoNode("demo-code-a", 420, 30, map[string]any{
			"kind":     "code",
			"label":    "方案 A：布隆过滤器",
			"code":     bloom,
			"language": "python",
			"source":   "user",
			"status":   "normal",
		}),
		demoNode("demo-code-b", 420, 290, map[string]any{
			"kind":     "code",
			"label":    "方案 B：空值缓存",
			"code":     nullCache,
			"language": "python",
			"source":   "user",
			"status":   "normal",
		}),
		demoNode("demo-code-evidence", 760, 30, map[string]any{
			"kind":         "text",
			"label":        "佐证：A 的内存与误判可控",
			"content":      "100 万元素、1% 误判率约需 1.19MB 位图与 7 次哈希；适合在入口层挡住绝大多数非法 key。",
			"source":       "agent",
			"status":       "normal",
			"ai_generated": true,
			"agent_name":   "代码评审官",
		}),
		demoNode("demo-code-counter", 760, 290, map[string]any{
			"kind":         "text",
			"label":        "反例：B 挡不住随机 key",
			"content":      "攻击者每次换新 key，空值缓存来不及命中，仍会持续查库；且大量空值占内存。",
			"source":       "agent",
			"status":       "normal",
			"ai_generated": true,
			"agent_name":   "代码评审官",
		}),
		demoNode("demo-code-conclusion", 1080, 160, map[string]any{
			"kind":    "text",
			"label":   "结论：先布隆，再短 TTL 空值兜底",
			"content": "## 组合策略\n\n布隆过滤器负责入口层拦截；对漏过的 key 使用 30s 空值缓存兜底；布隆过滤器定期重建，避免删除操作导致误判累积。",
			"source":  "user",
			"status":  "conclusion",
		}),
	}
	edges := []map[string]any{
		demoEdge("demo-code-edge-1", "demo-code-claim", "demo-code-a", "hypothesis"),
		demoEdge("demo-code-edge-2", "demo-code-claim", "demo-code-b", "hypothesis"),
		demoEdge("demo-code-edge-3", "demo-code-a", "demo-code-evidence", "evidence"),
		demoEdge("demo-code-edge-4", "demo-code-b", "demo-code-counter", "counter"),
		demoEdge("demo-code-edge-5", "demo-code-evidence", "demo-code-conclusion", "derive"),
		demoEdge("demo-code-edge-6", "demo-code-counter", "demo-code-conclusion", "counter"),
	}
	return demoCanvas{
		Title:       "Demo｜缓存穿透方案对比",
		Description: "两个代码方案并排推演：布隆过滤器 vs 空值缓存，各自补佐证与反例，最后组合成可落地策略。",
		Tags:        []string{"示例", "代码", "缓存"},
		Viewport:    map[string]float64{"x": 40, "y": 50, "zoom": 0.7},
		Nodes:       nodes,
		Edges:       edges,
	}
}
