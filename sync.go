package main

import (
	"github.com/rjeczalik/notify"
	"path/filepath"
	"os"
	"errors"
	"strings"
	"io"
	"strconv"
)

type Sync struct {
	containerFiler ContainerFiler
	sourceDir      string
	targetDir      string
	eventChan      chan notify.EventInfo
	fileToRenamed  string
	swapping       bool
	forceSync      bool
}

var ignoredExts []string = []string{"swp", "swx"}

func NewSync(containerFiler ContainerFiler, sourceDir, targetDir string) (*Sync, error) {

	fi, err := os.Stat(sourceDir)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, errors.New("You must pass a directory, not a file in source dir")
	}
	sourceDir = strings.TrimSuffix(sourceDir, "/")
	return &Sync{
		containerFiler: containerFiler,
		sourceDir: sourceDir,
		targetDir: targetDir,
		eventChan: make(chan notify.EventInfo, 50),
	}, nil
}

func (s *Sync) Run() error {
	err := s.syncFolder()
	if err != nil {
		return err
	}
	logger.Info("Start watching for change in folder '%s'\n", TruncatePath(s.sourceDir))
	if err := notify.Watch(s.sourceDir + "/...", s.eventChan, notify.Remove, notify.Create, notify.Write, notify.Rename); err != nil {
		return err
	}
	defer notify.Stop(s.eventChan)

	// Block until an event is received.
	for ei := range s.eventChan {
		if s.isIgnored(ei.Path()) {
			continue
		}
		logger.Info("Received event: '%s' for file '%s'", ei.Event().String(), TruncatePath(ei.Path()))
		err = s.action(ei)
		if err != nil {
			logger.Error("Event has errored: " + err.Error())
		}
	}
	return nil
}
func (s Sync) isIgnored(path string) bool {
	ext := filepath.Ext(path)
	for _, ignoredExt := range ignoredExts {
		if ext == "." + ignoredExt {
			return true
		}
	}
	_, err := strconv.Atoi(filepath.Base(path))
	if err == nil {
		return true
	}
	return false
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
	logger.Info("Synchronizing folder '%s' from the remote folder '%s' ...", TruncatePath(s.sourceDir), TruncatePath(s.targetDir))
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

func (s Sync) getFile(path string) (*os.File, os.FileInfo, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, stat, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, stat, err
	}
	return f, stat, nil
}
func (s *Sync) action(event notify.EventInfo) error {
	switch event.Event() {
	case notify.Write:
		return s.Write(event.Path())
	case notify.Create:
		return s.Create(event.Path())
	case notify.Remove:
		return s.Delete(event.Path())
	case notify.Rename:
		return s.Rename(event.Path())
	}
	return nil
}
func (s *Sync) Delete(path string) error {
	if s.swapping {
		s.swapping = false
		_, swappedFile := s.isSwapping(path)
		logger.Warning("File '%s' finished to swap, update sent.", TruncatePath(swappedFile))
		return s.Write(swappedFile)
	}
	return s.containerFiler.Delete(s.ToRemotePath(path))
}
func (s *Sync) Write(path string) error {
	if s.swapping {
		return nil
	}
	f, stat, err := s.getFile(path)
	if err != nil {
		return err
	}
	return s.containerFiler.CopyContent(f,
		stat.Size(),
		s.ToRemotePath(path),
		stat.Mode(),
	)
}
func (s *Sync) Create(path string) error {
	if s.swapping {
		return nil
	}
	isSwapping, swappingFile := s.isSwapping(path)
	if isSwapping {
		s.swapping = true
		logger.Warning("File '%s' is swapping, next events will be ignored.", TruncatePath(swappingFile))
		return nil
	}
	f, stat, err := s.getFile(path)
	if err != nil {
		return err
	}
	if stat.IsDir() {
		return s.containerFiler.CreateFolders(s.targetDir, s.TrimPath(path))
	} else {
		return s.containerFiler.CopyContent(f,
			stat.Size(),
			s.ToRemotePath(path),
			stat.Mode(),
		)
	}
}
func (s *Sync) Rename(path string) error {
	if s.swapping {
		return nil
	}
	exists, err := FileExists(path)
	if err != nil {
		return err
	}
	if !exists {
		isSwapping, swappingFile := s.isSwapping(path)
		if isSwapping {
			s.swapping = true
			logger.Warning("File '%s' is swapping, next events will be ignored.", TruncatePath(swappingFile))
			return nil
		}
	}
	if s.fileToRenamed == "" && !exists {
		return s.containerFiler.Delete(s.ToRemotePath(path))
	}
	if exists {
		s.fileToRenamed = path
		return nil
	}
	defer func() {
		s.fileToRenamed = ""
	}()
	return s.containerFiler.Rename(s.ToRemotePath(path), s.ToRemotePath(s.fileToRenamed))
}
func (s Sync) isSwapping(path string) (isSwapping bool, pathRenamed string) {
	return s.isSwappingWithLastState(path, false)
}
func (s Sync) isSwappingWithLastState(path string, state bool) (isSwapping bool, pathRenamed string) {
	pathRenamed = path
	ext := filepath.Ext(path)
	if ext == "." {
		isSwapping = false
		return
	}
	exists, _ := FileExists(path)
	if exists {
		isSwapping = state
		return
	}
	tempPath := strings.TrimSuffix(path, ext)
	return s.isSwappingWithLastState(tempPath + ext[:len(ext) - 1], true)
}
func (s Sync) TrimPath(path string) string {
	path = strings.TrimPrefix(path, s.sourceDir)
	path = filepath.ToSlash(path)
	path = strings.TrimPrefix(path, "/")
	return path
}
func (s Sync) ToRemotePath(path string) string {
	rmtPath := s.targetDir
	if !strings.HasSuffix(rmtPath, "/") {
		rmtPath += "/"
	}
	rmtPath = filepath.ToSlash(rmtPath)
	return rmtPath + s.TrimPath(path)
}
func (s *Sync) SetForceSync(forceSync bool) {
	s.forceSync = forceSync
}