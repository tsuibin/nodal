package migrate

import (
	"gorm.io/gorm"

	"github.com/nodal/nodal/backend/internal/model"
)

// builtinAgents 是四个内置 Agent 模板（PRD §4 F-08）。
// system_prompt 只描述角色职责与输出边界，不含任何用户数据。
var builtinAgents = []model.AgentTemplate{
	{
		Name:        "探索者",
		Role:        model.RoleExplorer,
		Description: "从当前节点发散更多可能性与分支",
		SystemPrompt: "你是 Nodal 的探索者。基于给定节点，提出 2-4 条值得继续推演的分支假设。" +
			"每条假设要指出它的关键前提，不要下结论，不要替用户决策。",
		IsBuiltin: true,
	},
	{
		Name:        "校验者",
		Role:        model.RoleVerifier,
		Description: "挑出逻辑漏洞、反例与不成立的隐含前提",
		SystemPrompt: "你是 Nodal 的校验者。针对给定节点寻找逻辑漏洞、反例与隐含前提。" +
			"每条质疑必须指出具体是哪一步推理不成立，给出可验证的检验方式。",
		IsBuiltin: true,
	},
	{
		Name:        "总结者",
		Role:        model.RoleSummarizer,
		Description: "把多条分支收拢成结构化结论",
		SystemPrompt: "你是 Nodal 的总结者。把画布上已有多条分支收拢为一份结构化结论：" +
			"共识、分歧、待验证项三部分，每部分不超过三条。",
		IsBuiltin: true,
	},
	{
		Name:        "构建者",
		Role:        model.RoleBuilder,
		Description: "把结论落成可直接使用的产物",
		SystemPrompt: "你是 Nodal 的构建者。把结论落成可直接使用的产物：代码、提纲、清单或方案骨架。" +
			"产物要能直接执行或直接落地，不要停在描述层。",
		IsBuiltin: true,
	},
}

// SeedBuiltinAgents 幂等写入内置模板。
func SeedBuiltinAgents(db *gorm.DB) error {
	for i := range builtinAgents {
		a := builtinAgents[i]
		// 已存在同名内置模板则跳过，不覆盖用户可能已经改过的 prompt
		var count int64
		if err := db.Model(&model.AgentTemplate{}).
			Where("is_builtin = true AND name = ?", a.Name).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		if err := db.Create(&a).Error; err != nil {
			return err
		}
	}
	return nil
}
