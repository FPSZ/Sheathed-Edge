package pathutil

import (
	"path/filepath"
	"runtime"
	"strings"
)

func NormalizeRuntimePath(path string) string {
	if path == "" {
		return path
	}
	if runtime.GOOS != "linux" {
		return filepath.Clean(path)
	}
	if len(path) >= 3 && path[1] == ':' && (path[2] == '\\' || path[2] == '/') {
		drive := strings.ToLower(path[:1])
		rest := strings.ReplaceAll(path[3:], "\\", "/")
		return filepath.Clean("/mnt/" + drive + "/" + rest)
	}
	return filepath.Clean(strings.ReplaceAll(path, "\\", "/"))
}

func ResolveSiblingPath(basePath, sibling string) string {
	return filepath.Clean(filepath.Join(filepath.Dir(basePath), sibling))
}
