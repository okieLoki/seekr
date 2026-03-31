package parser

import (
	"encoding/json"
	"strings"
)

func ExtractText(content string) string {
	s := strings.TrimSpace(content)
	if s == "" {
		return ""
	}

	if looksLikeJSON(s) {
		if out := extractJSON(s); out != "" {
			return out
		}
	}

	return content
}

func looksLikeJSON(s string) bool {
	return (strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")) ||
		(strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]"))
}

func extractJSON(s string) string {
	var raw interface{}
	if err := json.Unmarshal([]byte(s), &raw); err != nil {
		return ""
	}
	var parts []string
	flattenAny(raw, &parts)
	return strings.Join(parts, " ")
}

func flattenAny(v interface{}, out *[]string) {
	switch val := v.(type) {
	case string:
		if t := strings.TrimSpace(val); t != "" {
			*out = append(*out, t)
		}
	case map[string]interface{}:
		for _, child := range val {
			flattenAny(child, out)
		}
	case []interface{}:
		for _, child := range val {
			flattenAny(child, out)
		}
	}
}
