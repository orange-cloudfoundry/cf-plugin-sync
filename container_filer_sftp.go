package main

import (
	"io"
	"github.com/pkg/sftp"
	"strings"
	"os"
	"path/filepath"
	"github.com/cheggaaa/pb"
	"path"
	"fmt"
)

type ContainerFilerSftp struct {
	client *sftp.Client
	writer io.Writer
}

func NewContainerFiler(client *SecureClient) (ContainerFiler, error) {
	sftpClient, err := sftp.NewClient(client.Client())
	if err != nil {
		return nil, err
	}
	return &ContainerFilerSftp{
		client: sftpClient,
	}, nil
}
func (f ContainerFilerSftp) CopyRemoteFolder(sourceDir, targetDir string) error {
	targetDir = f.trimHome(targetDir)
	walker := f.client.Walk(targetDir)
	for walker.Step() {
		if walker.Path() == targetDir || walker.Stat().IsDir() {
			continue
		}
		if err := walker.Err(); err != nil {
			logger.Error(err.Error())
			continue
		}
		err := f.downloadFile(sourceDir, targetDir, walker.Path())
		if err != nil {
			logger.Error(err.Error())
		}
	}
	return nil
}
func (f *ContainerFilerSftp) downloadFile(sourceDir, targetDir, pathfile string) error {
	localPath := f.toLocalPath(sourceDir, targetDir, pathfile)
	directory := path.Dir(localPath)
	err := os.MkdirAll(directory, 0755)
	if err != nil {
		return err
	}
	stat, err := f.client.Stat(pathfile)
	if err != nil {
		return err
	}
	localFile, err := os.OpenFile(localPath, os.O_RDWR | os.O_CREATE | os.O_TRUNC, stat.Mode())
	if err != nil {
		return err
	}
	var remoteFile io.Reader
	remoteFile, err = f.client.Open(pathfile)
	if err != nil {
		return err
	}
	if f.writer != nil {
		bar := pb.New64(stat.Size()).SetUnits(pb.U_BYTES)
		bar.Output = f.writer
		bar.Prefix(fmt.Sprintf("Downloading file '%s' to '%s'...",
			TruncatePath(pathfile),
			filepath.FromSlash(TruncatePath(localPath))))
		bar.Start()
		remoteFile = bar.NewProxyReader(remoteFile)
	}
	_, err = io.Copy(localFile, remoteFile)
	if err != nil {
		return err
	}
	logger.Info(fmt.Sprintf("File '%s' downloaded to '%s'",
		TruncatePath(pathfile),
		filepath.FromSlash(TruncatePath(localPath))))
	return nil
}
func (f ContainerFilerSftp) toLocalPath(sourceDir, targetDir, pathfile string) string {
	if !strings.HasSuffix(sourceDir, string(os.PathSeparator)) {
		sourceDir += string(os.PathSeparator)
	}
	if !strings.HasSuffix(targetDir, "/") {
		targetDir += "/"
	}
	pathfile = strings.TrimPrefix(pathfile, targetDir)
	return sourceDir + filepath.FromSlash(pathfile)
}
func (f ContainerFilerSftp) trimHome(path string) string {
	return strings.TrimPrefix(path, "~/")
}
func (f ContainerFilerSftp) CopyContent(reader io.Reader, length int64, remotePath string, permissions os.FileMode) error {
	remotePath = f.trimHome(remotePath)
	if f.writer != nil {
		bar := pb.New64(length).SetUnits(pb.U_BYTES)
		bar.Output = f.writer
		bar.Prefix(fmt.Sprintf("Uploading file to '%s'...", TruncatePath(remotePath)))
		bar.Start()
		reader = bar.NewProxyReader(reader)
	}
	remoteFile, err := f.client.Create(remotePath)
	if err != nil {
		return err
	}
	_, err = io.Copy(remoteFile, reader)
	if err != nil {
		return err
	}
	return f.client.Chmod(remotePath, permissions)
}
func (f ContainerFilerSftp) CreateFolders(remotePath, dir string) error {
	remotePath = f.trimHome(remotePath)
	if !strings.HasSuffix(remotePath, "/") {
		remotePath = remotePath + "/"
	}
	logger.Info("Creating folder(s) '%s' in '%s' ...", dir, remotePath)
	dirs := strings.Split(dir, "/")
	for i := 0; i < len(dirs); i++ {
		dirToCreate := dirs[i]
		if dirToCreate == "" {
			continue
		}
		if i > 0 {
			remotePath = remotePath + strings.Join(dirs[:(i - 1)], "/") + "/"
		}
		err := f.client.Mkdir(remotePath + dirToCreate)
		if err != nil {
			return err
		}
	}
	logger.Info("Finished creating folder(s) '%s' in '%s'.", dir, remotePath)
	return nil
}
func (f ContainerFilerSftp) Delete(remotePath string) error {
	remotePath = f.trimHome(remotePath)
	logger.Info("Deleting path '%s' ...", remotePath)
	stat, err := f.client.Stat(remotePath)
	if err != nil {
		return err
	}
	if stat.IsDir() {
		err = f.client.RemoveDirectory(remotePath)
	} else {
		err = f.client.Remove(remotePath)
	}
	if err != nil {
		return err
	}
	logger.Info("Finished deleting path '%s'.", remotePath)
	return nil
}
func (f ContainerFilerSftp) Rename(srcRmtPath, trtRmtPath string) error {
	srcRmtPath = f.trimHome(srcRmtPath)
	trtRmtPath = f.trimHome(trtRmtPath)
	logger.Info("Moving path '%s' to '%s' ...", srcRmtPath, trtRmtPath)
	err := f.client.Rename(srcRmtPath, trtRmtPath)
	if err != nil {
		return err
	}
	logger.Info("Finished moving path '%s' to '%s' ...", srcRmtPath, trtRmtPath)
	return nil
}
func (f *ContainerFilerSftp) SetWriter(writer io.Writer) {
	f.writer = writer
}