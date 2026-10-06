package classifier

import (
	"context"
	"regexp"
)

// Result is the output of a classification.
type Result struct {
	Category       string  `json:"category"`
	BusinessModule string  `json:"business_module"`
	Confidence     float64 `json:"confidence"`
	Sentiment      string  `json:"sentiment"` // positive | neutral | negative
}

// Classifier classifies a feedback content string.
type Classifier interface {
	Classify(ctx context.Context, content string) (Result, error)
}

// parenPattern matches trailing parenthesized description like "会员(会员、订阅、续费)".
var parenPattern = regexp.MustCompile(`\([^)]*\)`)

// CleanBusinessModule strips parenthesized descriptions from the module name.
// Some LLMs may still return "名称(描述)" despite prompt instructions.
func CleanBusinessModule(name string) string {
	return parenPattern.ReplaceAllString(name, "")
}
