package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"gorm.io/datatypes"

	"github.com/nodal/nodal-server/internal/apperr"
	"github.com/nodal/nodal-server/internal/model"
)

// 画布文档限制。前端正常画布远小于这些上限，这里主要防脏数据和恶意载荷。
const (
	MaxCanvasBytes = 5 << 20 // 5MB（nodes + edges + viewport）
	MaxCanvasNodes = 2000
	MaxCanvasEdges = 5000
	MaxCanvasTags  = 10
	MaxTagRunes    = 20
	MaxTitleRunes  = 120
	MaxDescRunes   = 500
)

// canvasDocument 是一张画布的可持久化文档，结构与 React Flow 严格对齐：
//
//	{ "nodes": [...], "edges": [...], "viewport": { "x": 0, "y": 0, "zoom": 1 } }
//
// nodes / edges 原样保存，不裁剪任何 React Flow 字段（type、data、style、
// parentId、sourceHandle…），保证导出后前端可直接 setNodes/setEdges。
type canvasDocument struct {
	Nodes     json.RawMessage
	Edges     json.RawMessage
	Viewport  json.RawMessage
	NodeCount int
	EdgeCount int
}

type rfNode struct {
	ID       string `json:"id"`
	Position *struct {
		X *float64 `json:"x"`
		Y *float64 `json:"y"`
	} `json:"position"`
}

type rfEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
}

type rfViewport struct {
	X    *float64 `json:"x"`
	Y    *float64 `json:"y"`
	Zoom *float64 `json:"zoom"`
}

// normalizeDocument 校验并规范化画布文档。
// missing 语义：nil/空 视为空数组，便于「新建空白画布」。
func normalizeDocument(nodes, edges, viewport json.RawMessage) (*canvasDocument, error) {
	var err error

	nodes, err = normalizeJSONArray(nodes, "nodes")
	if err != nil {
		return nil, err
	}
	edges, err = normalizeJSONArray(edges, "edges")
	if err != nil {
		return nil, err
	}
	viewport, err = normalizeViewport(viewport)
	if err != nil {
		return nil, err
	}

	if len(nodes)+len(edges)+len(viewport) > MaxCanvasBytes {
		return nil, apperr.Newf(apperr.CodeInvalidParam, "画布数据过大，上限 %d MB", MaxCanvasBytes>>20)
	}

	// ---- 节点校验：id 必填且唯一，position 必须是数字 ----
	var rawNodes []json.RawMessage
	if err := json.Unmarshal(nodes, &rawNodes); err != nil {
		return nil, apperr.New(apperr.CodeInvalidParam, "nodes 必须是数组")
	}
	if len(rawNodes) > MaxCanvasNodes {
		return nil, apperr.Newf(apperr.CodeInvalidParam, "节点数量超过上限（%d）", MaxCanvasNodes)
	}

	nodeIDs := make(map[string]struct{}, len(rawNodes))
	for i, raw := range rawNodes {
		var n rfNode
		if err := json.Unmarshal(raw, &n); err != nil {
			return nil, nodeErr(i, "必须是对象")
		}
		id := strings.TrimSpace(n.ID)
		if id == "" {
			return nil, nodeErr(i, "缺少 id")
		}
		if _, dup := nodeIDs[id]; dup {
			return nil, nodeErr(i, fmt.Sprintf("id %q 重复", id))
		}
		nodeIDs[id] = struct{}{}
		if n.Position == nil || n.Position.X == nil || n.Position.Y == nil {
			return nil, nodeErr(i, "缺少 position.{x,y}")
		}
	}

	// ---- 连线校验：id 唯一，source/target 必须指向已存在的节点 ----
	var rawEdges []json.RawMessage
	if err := json.Unmarshal(edges, &rawEdges); err != nil {
		return nil, apperr.New(apperr.CodeInvalidParam, "edges 必须是数组")
	}
	if len(rawEdges) > MaxCanvasEdges {
		return nil, apperr.Newf(apperr.CodeInvalidParam, "连线数量超过上限（%d）", MaxCanvasEdges)
	}

	edgeIDs := make(map[string]struct{}, len(rawEdges))
	for i, raw := range rawEdges {
		var e rfEdge
		if err := json.Unmarshal(raw, &e); err != nil {
			return nil, edgeErr(i, "必须是对象")
		}
		if strings.TrimSpace(e.ID) == "" {
			return nil, edgeErr(i, "缺少 id")
		}
		if _, dup := edgeIDs[e.ID]; dup {
			return nil, edgeErr(i, fmt.Sprintf("id %q 重复", e.ID))
		}
		edgeIDs[e.ID] = struct{}{}
		if strings.TrimSpace(e.Source) == "" || strings.TrimSpace(e.Target) == "" {
			return nil, edgeErr(i, "缺少 source/target")
		}
		if _, ok := nodeIDs[e.Source]; !ok {
			return nil, edgeErr(i, fmt.Sprintf("source %q 不存在于 nodes", e.Source))
		}
		if _, ok := nodeIDs[e.Target]; !ok {
			return nil, edgeErr(i, fmt.Sprintf("target %q 不存在于 nodes", e.Target))
		}
	}

	return &canvasDocument{
		Nodes:     nodes,
		Edges:     edges,
		Viewport:  viewport,
		NodeCount: len(rawNodes),
		EdgeCount: len(rawEdges),
	}, nil
}

func normalizeJSONArray(raw json.RawMessage, field string) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return json.RawMessage("[]"), nil
	}
	if trimmed[0] != '[' {
		return nil, apperr.Newf(apperr.CodeInvalidParam, "%s 必须是数组", field)
	}
	// 做一次语法校验，避免把坏 JSON 写进库
	var arr []json.RawMessage
	if err := json.Unmarshal(trimmed, &arr); err != nil {
		return nil, apperr.Newf(apperr.CodeInvalidParam, "%s 不是合法 JSON 数组", field)
	}
	return trimmed, nil
}

func normalizeViewport(raw json.RawMessage) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return json.RawMessage(`{"x":0,"y":0,"zoom":1}`), nil
	}
	var vp rfViewport
	if err := json.Unmarshal(trimmed, &vp); err != nil {
		return nil, apperr.New(apperr.CodeInvalidParam, "viewport 必须是 {x, y, zoom} 对象")
	}
	if vp.X == nil || vp.Y == nil {
		return nil, apperr.New(apperr.CodeInvalidParam, "viewport 缺少 x/y")
	}
	if vp.Zoom != nil && (*vp.Zoom <= 0 || *vp.Zoom > 4) {
		return nil, apperr.New(apperr.CodeInvalidParam, "viewport.zoom 必须在 (0, 4] 之间")
	}
	// 只回写允许的三个字段，避免客户端塞入任意内容
	out := map[string]float64{"x": *vp.X, "y": *vp.Y, "zoom": 1}
	if vp.Zoom != nil {
		out["zoom"] = *vp.Zoom
	}
	data, _ := json.Marshal(out)
	return data, nil
}

func normalizeTags(tags []string) ([]string, error) {
	if len(tags) == 0 {
		return []string{}, nil
	}
	out := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if utf8.RuneCountInString(t) > MaxTagRunes {
			return nil, apperr.Newf(apperr.CodeInvalidParam, "标签「%s」过长（最多 %d 字）", t, MaxTagRunes)
		}
		key := strings.ToLower(t)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, t)
	}
	if len(out) > MaxCanvasTags {
		return nil, apperr.Newf(apperr.CodeInvalidParam, "标签最多 %d 个", MaxCanvasTags)
	}
	return out, nil
}

func encodeTags(tags []string) datatypes.JSON {
	if tags == nil {
		tags = []string{}
	}
	data, _ := json.Marshal(tags)
	return datatypes.JSON(data)
}

func decodeTags(raw datatypes.JSON) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var tags []string
	if err := json.Unmarshal(raw, &tags); err != nil || tags == nil {
		return []string{}
	}
	return tags
}

// normalizeVisibility 统一 is_public 与 share_mode：
// public 必公开、private 必不公开；传了 is_public=true 时至少升级到 public。
func normalizeVisibility(isPublic bool, mode string) (bool, string, error) {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case "":
		mode = model.ShareModePrivate
	case model.ShareModePrivate, model.ShareModeLink, model.ShareModePublic:
	default:
		return false, "", apperr.New(apperr.CodeInvalidParam, "share_mode 只能是 private / link / public")
	}

	if isPublic && mode != model.ShareModePublic {
		mode = model.ShareModePublic
	}
	switch mode {
	case model.ShareModePublic:
		isPublic = true
	case model.ShareModePrivate:
		isPublic = false
	}
	return isPublic, mode, nil
}

func nodeErr(i int, msg string) error {
	return apperr.Newf(apperr.CodeInvalidParam, "第 %d 个节点%s", i+1, msg)
}

func edgeErr(i int, msg string) error {
	return apperr.Newf(apperr.CodeInvalidParam, "第 %d 条连线%s", i+1, msg)
}

func trimTitle(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "未命名画布"
	}
	if utf8.RuneCountInString(s) > MaxTitleRunes {
		runes := []rune(s)
		return string(runes[:MaxTitleRunes])
	}
	return s
}
