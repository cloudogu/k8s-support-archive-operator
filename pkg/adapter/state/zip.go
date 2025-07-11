package state

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/config"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"slices"
)

const (
	stateFileFmt = "%s.json"
	statePath    = "/data/work"
	archivePath  = "/data/support-archives"
)

type State struct {
	Done               bool     `json:"done"`
	ExecutedCollectors []string `json:"executedCollectors"`
}

func (s *State) Add(collector string) {
	if slices.Contains(s.ExecutedCollectors, collector) {
		return
	}

	s.ExecutedCollectors = append(s.ExecutedCollectors, collector)
}

type zipCreator func(w io.Writer) Zipper

func NewZipWriter(w io.Writer) Zipper {
	return zip.NewWriter(w)
}

type ZipArchiver struct {
	filesystem                    volumeFs
	zipCreator                    zipCreator
	volumeDownloadServiceName     string
	volumeDownloadServiceProtocol string
	volumeDownloadServicePort     string
}

func NewArchiver(filesystem volumeFs, zipCreator zipCreator, config config.OperatorConfig) *ZipArchiver {
	return &ZipArchiver{
		filesystem:                    filesystem,
		zipCreator:                    zipCreator,
		volumeDownloadServiceName:     config.ArchiveVolumeDownloadServiceName,
		volumeDownloadServiceProtocol: config.ArchiveVolumeDownloadServiceProtocol,
		volumeDownloadServicePort:     config.ArchiveVolumeDownloadServicePort,
	}
}

func (a *ZipArchiver) Write(_ context.Context, _, name, namespace, zipFilePath string, writer func(w io.Writer) error) error {
	stateArchiveFilePath := filepath.Join(statePath, namespace, name, zipFilePath)
	err := a.filesystem.MkdirAll(filepath.Dir(stateArchiveFilePath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(stateArchiveFilePath), err)
	}

	open, err := a.filesystem.Create(stateArchiveFilePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", stateArchiveFilePath, err)
	}

	err = writer(open)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", zipFilePath, err)
	}

	return nil
}

func (a *ZipArchiver) WriteState(_ context.Context, name string, namespace string, stateName string) error {
	state, err := a.parseState(name, namespace)
	if err != nil {
		return err
	}

	state.Add(stateName)
	err = a.writeState(name, namespace, state)
	if err != nil {
		return err
	}

	return nil
}

// Finalize creates the final archive.
func (a *ZipArchiver) Finalize(ctx context.Context, name string, namespace string) error {
	logger := log.FromContext(ctx).WithName("ZipArchiver.Finalize")
	destinationPath := fmt.Sprintf("%s.zip", filepath.Join(archivePath, namespace, name))
	zipFile, err := a.openFile(destinationPath)
	if err != nil {
		return err
	}

	zipWriter := a.zipCreator(zipFile)
	defer func() {
		if closeErr := zipWriter.Close(); closeErr != nil {
			logger.Error(closeErr, "failed to close zip writer")
		}
		if closeErr := zipFile.Close(); closeErr != nil {
			logger.Error(closeErr, "failed to close zip file")
		}
	}()

	stateArchiveDir := filepath.Join(statePath, namespace, name)
	err = a.filesystem.WalkDir(stateArchiveDir, func(path string, d fs.DirEntry, err error) error {
		return a.CopyFileToArchive(zipWriter, stateArchiveDir, path, d, err)
	})

	if err != nil {
		return fmt.Errorf("failed to copy files to zip archive: %w", err)
	}

	err = a.filesystem.RemoveAll(stateArchiveDir)
	if err != nil {
		return fmt.Errorf("failed to remove state files %s: %w", stateArchiveDir, err)
	}

	state, err := a.parseState(name, namespace)
	if err != nil {
		return err
	}

	state.Done = true
	err = a.writeState(name, namespace, state)
	if err != nil {
		return err
	}

	return nil
}

func (a *ZipArchiver) CopyFileToArchive(zipper Zipper, stateArchiveDir, path string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if d.IsDir() {
		return nil
	}

	relativePath, err := filepath.Rel(stateArchiveDir, path)
	if err != nil {
		return fmt.Errorf("failed to get relative path for file %s: %w", path, err)
	}

	zipFileWriter, err := zipper.Create(relativePath)
	if err != nil {
		return fmt.Errorf("failed to create zip writer for file %s: %w", path, err)
	}

	open, err := a.filesystem.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", path, err)
	}

	_, err = a.filesystem.Copy(zipFileWriter, open)
	if err != nil {
		return fmt.Errorf("failed to copy file %s: %w", path, err)
	}

	return nil
}

// Read reads the actual state for the given support archive and returns the executed collectors and true if alle collectors are done.
func (a *ZipArchiver) Read(_ context.Context, name, namespace string) ([]string, bool, error) {
	state, err := a.parseState(name, namespace)
	if err != nil {
		return nil, false, err
	}

	return state.ExecutedCollectors, state.Done, nil
}

func (a *ZipArchiver) GetDownloadURL(_ context.Context, name, namespace string) string {
	return fmt.Sprintf("%s://%s.%s.svc.cluster.local:%s/%s/%s.zip", a.volumeDownloadServiceProtocol, a.volumeDownloadServiceName, namespace, a.volumeDownloadServicePort, namespace, name)
}

func (a *ZipArchiver) parseState(name, namespace string) (State, error) {
	path := fmt.Sprintf(stateFileFmt, filepath.Join(statePath, namespace, name))
	_, err := a.filesystem.Stat(path)
	if os.IsNotExist(err) {
		return State{}, nil
	}

	if err != nil {
		return State{}, fmt.Errorf("failed to stat state file %s: %w", path, err)
	}

	file, err := a.openFile(path)
	if err != nil {
		return State{}, err
	}

	data, err := a.filesystem.ReadAll(file)
	if err != nil {
		return State{}, fmt.Errorf("failed to read state file %s: %w", path, err)
	}

	state := &State{}
	err = json.Unmarshal(data, state)
	if err != nil {
		return State{}, fmt.Errorf("failed to unmarshal state file %s: %w", path, err)
	}

	return *state, nil
}

func (a *ZipArchiver) writeState(name, namespace string, state State) error {
	path := fmt.Sprintf(stateFileFmt, filepath.Join(statePath, namespace, name))

	marshal, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	err = a.createFileIfNotExists(path)
	if err != nil {
		return fmt.Errorf("failed to create state file %s: %w", path, err)
	}

	err = a.filesystem.WriteFile(path, marshal, 0644)
	if err != nil {
		return fmt.Errorf("failed to write state file %s: %w", path, err)
	}

	return nil
}

func (a *ZipArchiver) openFile(path string) (closableRWFile, error) {
	err := a.createFileIfNotExists(path)
	if err != nil {
		return nil, err
	}

	file, err := a.filesystem.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", path, err)
	}

	return file, nil
}

func (a *ZipArchiver) createFileIfNotExists(path string) error {
	_, err := a.filesystem.Stat(path)
	if os.IsNotExist(err) {
		dir := filepath.Dir(path)
		dirErr := a.filesystem.MkdirAll(dir, os.FileMode(0755))
		if dirErr != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, dirErr)
		}

		_, createErr := a.filesystem.Create(path)
		if createErr != nil {
			return fmt.Errorf("failed to create file: %w", createErr)
		}

		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to stat file %s: %w", path, err)
	}

	return nil
}

func (a *ZipArchiver) Clean(ctx context.Context, name, namespace string) error {
	logger := log.FromContext(ctx).WithName("FileCleaner").WithValues("SupportArchiveName", name)
	logger.Info("Remove support archive")

	archiveNamespaceDir := filepath.Join(archivePath, namespace)
	archiveFile := fmt.Sprintf("%s.zip", filepath.Join(archiveNamespaceDir, name))

	var multiErr []error
	if err := a.filesystem.Remove(archiveFile); err != nil {
		multiErr = append(multiErr, fmt.Errorf("failed to remove archive %s: %w", archiveFile, err))
	}

	if err := a.removeDirIfEmpty(archiveNamespaceDir); err != nil {
		multiErr = append(multiErr, err)
	}

	stateNamespaceDir := fmt.Sprintf("%s/%s", statePath, namespace)
	stateFile := fmt.Sprintf("%s/%s.json", stateNamespaceDir, name)
	if err := a.filesystem.Remove(stateFile); err != nil {
		multiErr = append(multiErr, fmt.Errorf("failed to remove state file %s: %w", stateFile, err))
	}
	if err := a.removeDirIfEmpty(stateNamespaceDir); err != nil {
		multiErr = append(multiErr, err)
	}

	return errors.Join(multiErr...)
}

func (a *ZipArchiver) removeDirIfEmpty(path string) error {
	dirEntries, err := a.filesystem.ReadDir(path)
	if err != nil {
		return fmt.Errorf("error reading dir %q: %w", path, err)
	}

	if len(dirEntries) != 0 {
		return nil
	}

	err = a.filesystem.Remove(path)
	if err != nil {
		return fmt.Errorf("error removing empty dir %q: %w", path, err)
	}

	return nil
}
