package main

import (
	"os"
	"path/filepath"
	"strings"
	"io/ioutil"
	"github.com/monochromegane/go-gitignore"
)

const IGNORE_FILENAME = ".syncignore"

type SyncIgnore struct {
	rootDir       string
	base          string
	ignoreMatcher gitignore.IgnoreMatcher
}

func NewSyncIgnore(rootDir, base string) (*SyncIgnore, error) {
	syncIgnore := &SyncIgnore{
		rootDir: rootDir,
		base: base,
	}
	err := syncIgnore.Load()
	if err != nil {
		return nil, err
	}
	return syncIgnore, nil
}
func (i SyncIgnore) getFile() (*os.File, error) {
	rootDir, err := filepath.Abs(i.rootDir)
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(rootDir, string(os.PathSeparator)) {
		rootDir += string(os.PathSeparator)
	}
	pathIgnoreFile := rootDir + IGNORE_FILENAME
	exists, err := FileExists(pathIgnoreFile)
	if err != nil {
		return nil, err
	}
	if exists {
		return os.Open(pathIgnoreFile)
	}
	exists, err = FileExists(IGNORE_FILENAME)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	f, err := os.Create(pathIgnoreFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := ioutil.ReadFile(IGNORE_FILENAME)
	if err != nil {
		return nil, err
	}
	_, err = f.WriteString(string(b))
	if err != nil {
		return nil, err
	}
	return os.Open(IGNORE_FILENAME)
}

func (i *SyncIgnore) Load() error {
	f, err := i.getFile()
	if err != nil {
		return err
	}
	if f == nil {
		return nil
	}
	defer f.Close()
	i.ignoreMatcher = gitignore.NewGitIgnoreFromReader(i.base, f)
	return nil
}
func (i SyncIgnore) Match(pathfile string, isDir bool) bool {
	if i.ignoreMatcher == nil {
		return false
	}
	return i.ignoreMatcher.Match(pathfile, isDir)
}