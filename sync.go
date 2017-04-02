package main

import (
	"github.com/rjeczalik/notify"
	"path/filepath"
	"os"
	"errors"
	"strings"
	"strconv"
	"io"
)

type Sync struct {
	containerFiler *ContainerFiler
	sourceDir      string
	targetDir      string
	eventChan      chan notify.EventInfo
	fileToRenamed  string
	forceSync      bool
}

func NewSync(containerFiler *ContainerFiler, sourceDir, targetDir string) (*Sync, error) {
	dir, err := filepath.Abs(sourceDir)
	if err != nil {
		return nil, err
	}
	dirExists, err := fileExists(dir)
	if err != nil {
		return nil, err
	}
	if !dirExists {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return nil, err
		}
	}
	fi, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, errors.New("You must pass a directory, not a file in source dir")
	}
	dir = strings.TrimSuffix(dir, "/")
	return &Sync{
		containerFiler: containerFiler,
		sourceDir: dir,
		targetDir: targetDir,
		eventChan: make(chan notify.EventInfo, 1),
	}, nil
}

func (s *Sync) Run() error {
	err := s.syncFolder()
	if err != nil {
		return err
	}
	logger.Info("Start watching for change in folder '%s'\n", s.sourceDir)
	if err := notify.Watch(s.sourceDir + "/...", s.eventChan, notify.Remove, notify.Create, notify.Write, notify.Rename); err != nil {
		return err
	}
	defer notify.Stop(s.eventChan)

	// Block until an event is received.
	for ei := range s.eventChan {
		logger.Info("Received event: '%s' for file '%s'", ei.Event().String(), ei.Path())
		err = s.action(ei)
		if err != nil {
			logger.Error("Event has errored: " + err.Error())
		}
	}
	return nil
}
func (s *Sync) syncFolder() error {
	dirIsEmpty, err := s.DirIsEmpty(s.sourceDir)
	if err != nil {
		return err
	}
	if !dirIsEmpty && !s.forceSync {
		logger.Info("No need to synchronize from remote, directory not empty.")
		return nil
	}
	logger.Info("Synchronizing folder '%s' from the remote folder '%s' ...", s.sourceDir, s.targetDir)
	err = s.containerFiler.CopyRemoteFolder(s.sourceDir, s.targetDir)
	if err != nil {
		return err
	}
	logger.Info("Synchronization finished.\n")
	return nil
}
func (s Sync) DirIsEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}
func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}
func (s Sync) getFile(event notify.EventInfo) (*os.File, os.FileInfo, error) {
	stat, err := os.Stat(event.Path())
	if err != nil {
		return nil, stat, err
	}
	f, err := os.Open(event.Path())
	if err != nil {
		return nil, stat, err
	}
	return f, stat, nil
}
func (s *Sync) action(event notify.EventInfo) error {
	switch event.Event() {
	case notify.Write:
		return s.Write(event)
	case notify.Create:
		return s.Create(event)
	case notify.Remove:
		return s.containerFiler.Delete(s.ToRemotePath(event.Path()))
	case notify.Rename:
		return s.Rename(event)
	}
	return nil
}
func (s *Sync) Write(event notify.EventInfo) error {
	f, stat, err := s.getFile(event)
	if err != nil {
		return err
	}
	return s.containerFiler.CopyContent(f,
		stat.Size(),
		s.ToRemotePath(event.Path()),
		"0" + strconv.FormatUint(uint64(stat.Mode()), 8),
	)
}
func (s *Sync) Create(event notify.EventInfo) error {
	f, stat, err := s.getFile(event)
	if err != nil {
		return err
	}
	if stat.IsDir() {
		return s.containerFiler.CreateFolders(s.targetDir, s.TrimPath(event.Path()))
	} else {
		return s.containerFiler.CopyContent(f,
			stat.Size(),
			s.ToRemotePath(event.Path()),
			"0" + strconv.FormatUint(uint64(stat.Mode()), 8),
		)
	}
}
func (s *Sync) Rename(event notify.EventInfo) error {
	exists, err := fileExists(event.Path())
	if err != nil {
		return err
	}
	if s.fileToRenamed == "" && !exists {
		return s.containerFiler.Delete(s.ToRemotePath(event.Path()))
	}
	if s.fileToRenamed == "" {
		s.fileToRenamed = event.Path()
		return nil
	}
	defer func() {
		s.fileToRenamed = ""
	}()
	return s.containerFiler.Rename(s.ToRemotePath(event.Path()), s.ToRemotePath(s.fileToRenamed))
}
func (s Sync) TrimPath(path string) string {
	path = strings.TrimPrefix(path, s.sourceDir)
	path = strings.Replace(path, "\\", "/", -1)
	path = strings.TrimPrefix(path, "/")
	return path
}
func (s Sync) ToRemotePath(path string) string {
	rmtPath := filepath.Dir(s.targetDir)
	if !strings.HasSuffix(rmtPath, "/") {
		rmtPath += "/"
	}
	rmtPath = strings.Replace(rmtPath, "\\", "/", -1)
	return rmtPath + s.TrimPath(path)
}
func (s *Sync) SetForceSync(forceSync bool) {
	s.forceSync = forceSync
}