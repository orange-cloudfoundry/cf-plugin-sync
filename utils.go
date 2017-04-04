package main

import (
	"path/filepath"
	"strings"
)

func TruncatePath(path string) string {
	path = filepath.ToSlash(path)
	splittedPath := strings.Split(path, "/")
	if len(splittedPath) <= 3 {
		return path
	}
	return "..." + strings.Join(splittedPath[len(splittedPath) - 3:], "/")
}