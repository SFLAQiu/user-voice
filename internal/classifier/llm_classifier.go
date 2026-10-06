package classifier

import (
	"encoding/json"
	"fmt"
	"strings"
)

// parseResult extracts a classification Result from raw LLM response text.
func parseResult(raw string) (Result, error) {
	s := raw
	if i := strings.Index(s, "{"); i >= 0 {
		if j := strings.LastIndex(s, "}"); j >= i {
			s = s[i : j+1]
		}
	}
	var r Result
	if err := json.Unmarshal([]byte(s), &r); err != nil {
		return Result{}, fmt.Errorf("parse llm output %q: %w", raw, err)
	}
	if r.Sentiment == "" {
		r.Sentiment = "neutral"
	}
	return r, nil
}
