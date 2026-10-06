package classifier

import (
	"fmt"
	"strings"
)

// BuildPrompt constructs the final prompt from a template and dynamic lists.
func BuildPrompt(template, categories, businessModules, content string) string {
	p := strings.ReplaceAll(template, "{categories}", categories)
	p = strings.ReplaceAll(p, "{business_modules}", businessModules)
	p = strings.ReplaceAll(p, "{content}", content)
	return p
}

// FormatCategories joins enabled category names into a display string for the LLM prompt.
func FormatCategories(names []string) string {
	return strings.Join(names, "、")
}

// FormatBusinessModules builds the module list for the LLM prompt.
// Names are listed as selectable options; descriptions appear as parenthesized
// explanations so the LLM understands context but returns only the name.
func FormatBusinessModules(items []ModuleItem) string {
	names := make([]string, 0, len(items))
	for _, m := range items {
		names = append(names, m.Name)
	}
	nameList := strings.Join(names, "、")

	descParts := make([]string, 0, len(items))
	for _, m := range items {
		if m.Description != "" {
			descParts = append(descParts, fmt.Sprintf("%s：%s", m.Name, m.Description))
		}
	}
	if len(descParts) == 0 {
		return nameList
	}
	return fmt.Sprintf("%s（%s）", nameList, strings.Join(descParts, "；"))
}

// ModuleItem represents a business module with its optional description.
type ModuleItem struct {
	Name        string
	Description string
}

// FallbackPromptTemplate is used when no DB config is found.
const FallbackPromptTemplate = `你是产品反馈分类助手。请根据反馈内容输出严格 JSON，不要输出任何其他文字：
{
  "category": "<分类>",
  "business_module": "<业务模块>",
  "confidence": <0到1的小数>,
  "sentiment": "positive|neutral|negative"
}
可选分类：{categories}
可选业务模块：{business_modules}
注意：business_module 字段只能填写模块名称，不要包含括号中的说明文字。

反馈内容：{content}`