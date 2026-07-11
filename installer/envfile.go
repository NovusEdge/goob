package main

import (
	"bufio"
	"bytes"
	"strings"
)

// EnvLine represents a line in a .env file.
type EnvLine struct {
	Key     string
	Value   string
	Raw     string // original line (for comments, blanks)
	IsKV    bool   // true if this is a key=value line
	Export  bool   // true if prefixed with "export "
}

// ParseEnv parses a .env file, preserving comments and ordering.
func ParseEnv(data []byte) []EnvLine {
	var lines []EnvLine
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, parseEnvLine(line))
	}
	return lines
}

func parseEnvLine(line string) EnvLine {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return EnvLine{Raw: line}
	}
	export := false
	s := trimmed
	if strings.HasPrefix(s, "export ") {
		export = true
		s = strings.TrimPrefix(s, "export ")
	}
	idx := strings.Index(s, "=")
	if idx < 0 {
		return EnvLine{Raw: line}
	}
	key := strings.TrimSpace(s[:idx])
	val := s[idx+1:]
	// strip surrounding quotes
	val = strings.Trim(val, `"'`)
	return EnvLine{Key: key, Value: val, Raw: line, IsKV: true, Export: export}
}

// MergeEnv updates lines with new values, preserving unmanaged keys.
// Only keys in updates are modified; empty values are skipped (preserve existing).
func MergeEnv(lines []EnvLine, updates map[string]string) []EnvLine {
	seen := make(map[string]bool)
	result := make([]EnvLine, 0, len(lines))
	for _, l := range lines {
		if l.IsKV {
			if newVal, ok := updates[l.Key]; ok {
				seen[l.Key] = true
				if newVal == "" {
					// keep existing
					result = append(result, l)
				} else {
					// update
					result = append(result, EnvLine{
						Key: l.Key, Value: newVal, IsKV: true, Export: l.Export,
					})
				}
				continue
			}
		}
		result = append(result, l)
	}
	// append new keys
	for k, v := range updates {
		if !seen[k] && v != "" {
			result = append(result, EnvLine{Key: k, Value: v, IsKV: true})
		}
	}
	return result
}

// SerializeEnv converts lines back to .env format.
func SerializeEnv(lines []EnvLine) []byte {
	var buf bytes.Buffer
	for _, l := range lines {
		if l.IsKV {
			if l.Export {
				buf.WriteString("export ")
			}
			buf.WriteString(l.Key)
			buf.WriteString("=")
			// quote if contains spaces or special chars
			if strings.ContainsAny(l.Value, " \t\"'") {
				buf.WriteString(`"`)
				buf.WriteString(strings.ReplaceAll(l.Value, `"`, `\"`))
				buf.WriteString(`"`)
			} else {
				buf.WriteString(l.Value)
			}
			buf.WriteString("\n")
		} else {
			buf.WriteString(l.Raw)
			buf.WriteString("\n")
		}
	}
	return buf.Bytes()
}
