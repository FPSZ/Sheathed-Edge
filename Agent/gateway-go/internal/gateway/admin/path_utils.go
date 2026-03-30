package admin

import (
	"path"
	"path/filepath"
	"strings"
	"unicode"
)

func isWindowsAbsolutePath(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) >= 3 && unicode.IsLetter(rune(value[0])) && value[1] == ':' && (value[2] == '\\' || value[2] == '/') {
		return true
	}
	return strings.HasPrefix(value, `\\`) || strings.HasPrefix(value, `//`)
}

func isPortableAbsolutePath(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	return filepath.IsAbs(value) || isWindowsAbsolutePath(value)
}

func cleanPortablePath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if isWindowsAbsolutePath(value) {
		return strings.ReplaceAll(path.Clean(strings.ReplaceAll(value, "\\", "/")), "/", `\`)
	}
	return filepath.Clean(value)
}

func normalizedPortablePathKey(value string) string {
	normalized := cleanPortablePath(value)
	normalized = strings.ReplaceAll(normalized, "\\", "/")
	normalized = strings.TrimRight(normalized, "/")
	return strings.ToLower(normalized)
}
