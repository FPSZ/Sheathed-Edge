package admin

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

func readRecentEntries(dir string, limit int, filter func(map[string]any) bool) ([]map[string]any, error) {
	if dir == "" || limit <= 0 {
		return nil, nil
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if err != nil {
		return nil, err
	}
	sort.Sort(sort.Reverse(sort.StringSlice(files)))

	var entries []map[string]any
	for _, path := range files {
		fileEntries, err := readJSONLFile(path, filter)
		if err != nil {
			continue
		}
		for i := len(fileEntries) - 1; i >= 0; i-- {
			entries = append(entries, fileEntries[i])
			if len(entries) >= limit {
				return entries, nil
			}
		}
	}
	return entries, nil
}

func readJSONLFile(path string, filter func(map[string]any) bool) ([]map[string]any, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)

	var out []map[string]any
	for scanner.Scan() {
		line := scanner.Bytes()
		var item map[string]any
		if err := json.Unmarshal(line, &item); err != nil {
			continue
		}
		if filter != nil && !filter(item) {
			continue
		}
		out = append(out, item)
	}
	return out, scanner.Err()
}
