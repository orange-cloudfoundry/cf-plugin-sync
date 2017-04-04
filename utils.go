package main

import (
	"path/filepath"
	"strings"
	"os"
)

func TruncatePath(path string) string {
	path = filepath.ToSlash(path)
	splittedPath := strings.Split(path, "/")
	if len(splittedPath) <= 3 {
		return path
	}
	return "..." + strings.Join(splittedPath[len(splittedPath) - 3:], "/")
}

func FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}