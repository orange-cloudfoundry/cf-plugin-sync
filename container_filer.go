package main

import (
	"io"
	"os"
)

type ContainerFiler interface {
	CopyRemoteFolder(sourceDir, targetDir string) error
	CopyContent(reader io.Reader, length int64, remotePath string, permissions os.FileMode) error
	CreateFolders(remotePath, dir string) error
	Delete(remotePath string) error
	Rename(srcRmtPath, trtRmtPath string) error
	SetWriter(writer io.Writer)
}


