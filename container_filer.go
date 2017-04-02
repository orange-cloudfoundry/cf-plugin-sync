package main

import (
	"io"
	"path"
	"fmt"
	"os/exec"
	"os"
	"strings"
	"github.com/cheggaaa/pb"
)

const (
	SCP_BIN_PATH = "/usr/bin/scp"
	SCP_BIN_PATH_KEY_ENV_VAR = "SCP_PATH"
)

type ContainerFiler struct {
	client    *SecureClient
	OutWriter io.Writer
}

func NewContainerFiler(client *SecureClient) *ContainerFiler {
	return &ContainerFiler{
		client: client,
	}
}
func (f ContainerFiler) CopyRemoteFolder(sourceDir, targetDir string) error {
	sess, err := f.client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()
	cmd := exec.Command(f.getScpBinPath(), "-t", "-r", "-v", sourceDir)
	if f.OutWriter != nil {
		cmd.Stderr = f.OutWriter
		sess.Stderr = f.OutWriter
	}
	outCmd, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	outSess, err := sess.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stdin = outSess
	sess.Stdin = outCmd
	err = sess.Start("/usr/bin/scp -qrf " + targetDir)
	if err != nil {
		return err
	}
	return cmd.Run()
}
func (f ContainerFiler) getScpBinPath() string {
	if os.Getenv(SCP_BIN_PATH_KEY_ENV_VAR) != "" {
		return os.Getenv(SCP_BIN_PATH_KEY_ENV_VAR)
	}
	return SCP_BIN_PATH
}

func (f ContainerFiler) CopyContent(reader io.Reader, length int64, remotePath string, permissions string) error {
	sess, err := f.client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()
	if f.OutWriter != nil {
		sess.Stderr = f.OutWriter
		sess.Stdout = f.OutWriter
		bar := pb.New64(length).SetUnits(pb.U_BYTES)
		bar.Output = f.OutWriter
		bar.Prefix("Uploading file to '" + remotePath + "'")
		bar.Start()
		reader = bar.NewProxyReader(reader)
	}
	w, err := sess.StdinPipe()
	if err != nil {
		return err
	}
	defer w.Close()
	filename := path.Base(remotePath)
	directory := path.Dir(remotePath)

	err = sess.Start("/usr/bin/scp -tq " + directory)
	if err != nil {
		return err
	}

	fmt.Fprintln(w, "C" + permissions, length, filename)
	io.Copy(w, reader)
	fmt.Fprintln(w, "\n\x00")
	sess.Wait()
	return nil
}
func (f ContainerFiler) CreateFolders(remotePath, dir string) error {
	if !strings.HasSuffix(remotePath, "/") {
		remotePath += "/"
	}
	dirs := strings.Split(dir, "/")
	for i := 0; i < len(dirs); i++ {
		dirToCreate := dirs[i]
		if dirToCreate == "" {
			continue
		}
		if i > 0 {
			remotePath = remotePath + strings.Join(dirs[:(i - 1)], "/") + "/"
		}
		err := f.createFolder(remotePath, dirToCreate)
		if err != nil {
			return err
		}
	}
	return nil
}
func (f ContainerFiler) createFolder(remotePath, directory string) error {
	logger.Info("Creating folder '%s' in remote '%s' ...", directory, remotePath)
	sess, err := f.client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()
	if f.OutWriter != nil {
		sess.Stderr = f.OutWriter
		sess.Stdout = f.OutWriter
	}
	w, err := sess.StdinPipe()
	if err != nil {
		return err
	}
	defer w.Close()
	err = sess.Start("/usr/bin/scp -trq " + remotePath)
	if err != nil {
		return err
	}
	fmt.Fprintln(w, "D0755", 0, directory)
	fmt.Fprintln(w, "\x00")
	sess.Wait()
	logger.Info("Finished creating folder '%s' in remote '%s'.", directory, remotePath)
	return nil
}
func (f ContainerFiler) Delete(remotePath string) error {
	logger.Info("Deleting path '%s' ...", remotePath)
	sess, err := f.client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()
	err = sess.Run("/bin/rm -Rf " + remotePath)
	if err != nil {
		return err
	}
	logger.Info("Finished deleting path '%s'.", remotePath)
	return nil
}
func (f ContainerFiler) Rename(srcRmtPath, trtRmtPath string) error {
	logger.Info("Moving path '%s' to '%s' ...", srcRmtPath, trtRmtPath)
	sess, err := f.client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()
	err = sess.Run("/bin/mv " + srcRmtPath + " " + trtRmtPath)
	if err != nil {
		return err
	}
	logger.Info("Finished moving path '%s' to '%s' ...", srcRmtPath, trtRmtPath)
	return nil
}