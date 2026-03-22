package parser

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"strings"

	"github.com/BurntSushi/toml"
	"golang.org/x/net/html"
	"gopkg.in/yaml.v3"
)

func ExtractText(content string) string {
	s := strings.TrimSpace(content)
	if s == "" {
		return ""
	}

	// JSON
	if looksLikeJSON(s) {
		if out := extractJSON(s); out != "" {
			return out
		}
	}

	// XML / HTML (both start with <)
	if strings.HasPrefix(s, "<") {
		if out := extractHTML(s); out != "" {
			return out
		}
		if out := extractXML(s); out != "" {
			return out
		}
	}

	// TOML (has key = value lines before YAML might)
	if looksLikeTOML(s) {
		if out := extractTOML(s); out != "" {
			return out
		}
	}

	// YAML
	if looksLikeYAML(s) {
		if out := extractYAML(s); out != "" {
			return out
		}
	}

	// Plain text fallback
	return content
}

func looksLikeJSON(s string) bool {
	return (strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")) ||
		(strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]"))
}

func looksLikeTOML(s string) bool {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// TOML has key = value or [section] lines
		if strings.Contains(line, " = ") || (strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]")) {
			return true
		}
		break
	}
	return false
}

func looksLikeYAML(s string) bool {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return strings.Contains(line, ": ")
	}
	return false
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

func extractYAML(s string) string {
	var raw interface{}
	if err := yaml.Unmarshal([]byte(s), &raw); err != nil {
		return ""
	}
	var parts []string
	flattenAny(raw, &parts)
	return strings.Join(parts, " ")
}

func extractTOML(s string) string {
	var raw map[string]interface{}
	if _, err := toml.Decode(s, &raw); err != nil {
		return ""
	}
	var parts []string
	flattenAny(raw, &parts)
	return strings.Join(parts, " ")
}

func extractHTML(s string) string {
	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		return ""
	}
	var parts []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			t := strings.TrimSpace(n.Data)
			if t != "" {
				parts = append(parts, t)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return strings.Join(parts, " ")
}

func extractXML(s string) string {
	decoder := xml.NewDecoder(strings.NewReader(s))
	var parts []string
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return ""
		}
		if cd, ok := tok.(xml.CharData); ok {
			t := strings.TrimSpace(string(cd))
			if t != "" {
				parts = append(parts, t)
			}
		}
	}
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
	// YAML/TOML can produce map[interface{}]interface{}
	case map[interface{}]interface{}:
		for _, child := range val {
			flattenAny(child, out)
		}
	}
	// Numbers, booleans, nil — intentionally skipped; not useful for text search
}
