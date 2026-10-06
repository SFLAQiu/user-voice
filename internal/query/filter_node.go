package query

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FilterNode supports recursive AND/OR nesting for filter conditions.
// A leaf node has field+op+value; a group node has type+children.
type FilterNode struct {
	Type     string        `json:"type"`     // "and" | "or" | "" (empty = leaf node)
	Field    string        `json:"field"`    // leaf: field name
	Op       string        `json:"op"`       // leaf: operator
	Value    interface{}   `json:"value"`    // leaf: value
	Children []FilterNode  `json:"children"` // group: child nodes
}

// IsLeaf returns true if the node is a leaf (field+op+value).
func (n FilterNode) IsLeaf() bool {
	return n.Field != ""
}

// IsGroup returns true if the node is a group (AND/OR with children).
func (n FilterNode) IsGroup() bool {
	return n.Type == "and" || n.Type == "or"
}

// NormalizeFilterTree parses raw JSON into a FilterNode tree.
// Supports both the new tree format and the legacy flat []Filter array.
func NormalizeFilterTree(raw json.RawMessage) (*FilterNode, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	// Try tree format first
	var node FilterNode
	if err := json.Unmarshal(raw, &node); err == nil && node.IsGroup() {
		return &node, nil
	}

	// Try legacy flat []Filter format
	var filters []Filter
	if err := json.Unmarshal(raw, &filters); err != nil {
		return nil, fmt.Errorf("parse filter tree: %w", err)
	}
	if len(filters) == 0 {
		return nil, nil
	}
	children := make([]FilterNode, len(filters))
	for i, f := range filters {
		children[i] = FilterNode{Field: f.Field, Op: f.Op, Value: f.Value}
	}
	return &FilterNode{Type: "and", Children: children}, nil
}

// BuildFilterNodeSQL recursively generates a WHERE clause from a FilterNode tree.
func BuildFilterNodeSQL(node FilterNode) (string, []interface{}, error) {
	if node.IsLeaf() {
		return buildLeafSQL(node)
	}
	if node.IsGroup() && len(node.Children) > 0 {
		return buildGroupSQL(node)
	}
	return "", nil, nil
}

func buildLeafSQL(n FilterNode) (string, []interface{}, error) {
	col, ok := allowedFields[n.Field]
	if !ok {
		return "", nil, fmt.Errorf("unknown filter field: %s", n.Field)
	}
	op, ok := allowedOps[strings.ToLower(n.Op)]
	if !ok {
		return "", nil, fmt.Errorf("unsupported op: %s", n.Op)
	}
	switch strings.ToLower(n.Op) {
	case "in", "not in":
		vals, ok := toSlice(n.Value)
		if !ok {
			return "", nil, fmt.Errorf("filter %s %s expects array value", n.Field, n.Op)
		}
		placeholders := strings.Repeat("?,", len(vals))
		placeholders = placeholders[:len(placeholders)-1]
		return fmt.Sprintf("%s %s (%s)", col, op, placeholders), vals, nil
	default:
		return col + " " + op + " ?", []interface{}{n.Value}, nil
	}
}

func buildGroupSQL(n FilterNode) (string, []interface{}, error) {
	var clauses []string
	var args []interface{}
	for _, child := range n.Children {
		sql, cargs, err := BuildFilterNodeSQL(child)
		if err != nil {
			return "", nil, err
		}
		if sql == "" {
			continue
		}
		clauses = append(clauses, sql)
		args = append(args, cargs...)
	}
	if len(clauses) == 0 {
		return "", nil, nil
	}
	joiner := " AND "
	if n.Type == "or" {
		joiner = " OR "
	}
	return "(" + strings.Join(clauses, joiner) + ")", args, nil
}

// BuildFilterNodeSQLWithFields 递归生成 WHERE 子句，字段名用提供的 fields 映射校验。
// 供 preagg 查询使用 preAggAllowedFields，而非全局 allowedFields（feedbacks 表列）。
func BuildFilterNodeSQLWithFields(node FilterNode, fields map[string]string) (string, []interface{}, error) {
	if node.IsLeaf() {
		return buildLeafSQLWithFields(node, fields)
	}
	if node.IsGroup() && len(node.Children) > 0 {
		return buildGroupSQLWithFields(node, fields)
	}
	return "", nil, nil
}

func buildLeafSQLWithFields(n FilterNode, fields map[string]string) (string, []interface{}, error) {
	col, ok := fields[n.Field]
	if !ok {
		return "", nil, fmt.Errorf("field %s not available in this context", n.Field)
	}
	op, ok := allowedOps[strings.ToLower(n.Op)]
	if !ok {
		return "", nil, fmt.Errorf("unsupported op: %s", n.Op)
	}
	switch strings.ToLower(n.Op) {
	case "in", "not in":
		vals, ok := toSlice(n.Value)
		if !ok {
			return "", nil, fmt.Errorf("filter %s %s expects array value", n.Field, n.Op)
		}
		placeholders := strings.Repeat("?,", len(vals))
		placeholders = placeholders[:len(placeholders)-1]
		return fmt.Sprintf("%s %s (%s)", col, op, placeholders), vals, nil
	default:
		return col + " " + op + " ?", []interface{}{n.Value}, nil
	}
}

func buildGroupSQLWithFields(n FilterNode, fields map[string]string) (string, []interface{}, error) {
	var clauses []string
	var args []interface{}
	for _, child := range n.Children {
		sql, cargs, err := BuildFilterNodeSQLWithFields(child, fields)
		if err != nil {
			return "", nil, err
		}
		if sql == "" {
			continue
		}
		clauses = append(clauses, sql)
		args = append(args, cargs...)
	}
	if len(clauses) == 0 {
		return "", nil, nil
	}
	joiner := " AND "
	if n.Type == "or" {
		joiner = " OR "
	}
	return "(" + strings.Join(clauses, joiner) + ")", args, nil
}

// trackPreAggFilters 递归遍历 FilterNode 树，检测 category/business_module/sentiment 引用，
// 用于触发 category_status IN (1, 3) 自动过滤。
func trackPreAggFilters(node FilterNode, hasCategory *bool, hasSentiment *bool) {
	if node.IsLeaf() {
		if node.Field == "category" || node.Field == "business_module" {
			*hasCategory = true
		}
		if node.Field == "sentiment" {
			*hasSentiment = true
		}
	} else if node.IsGroup() {
		for _, child := range node.Children {
			trackPreAggFilters(child, hasCategory, hasSentiment)
		}
	}
}