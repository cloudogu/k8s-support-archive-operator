package state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/config"
	"io"
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
	ExecutedCollectors []string `json:"executedCollectors"`
}

func (s *State) Add(collector string) {
	if slices.Contains(s.ExecutedCollectors, collector) {
		return
	}

	s.ExecutedCollectors = append(s.ExecutedCollectors, collector)
}

type ZipArchiver struct {
	filesystem                           volumeFs
	zipCreator                           zipCreator
	archiveVolumeDownloadServiceName     string
	archiveVolumeDownloadServiceProtocol string
	archiveVolumeDownloadServicePort     string
}

func NewArchiver(filesystem volumeFs, zipCreator zipCreator, config config.OperatorConfig) *ZipArchiver {
	return &ZipArchiver{
		filesystem:                           filesystem,
		zipCreator:                           zipCreator,
		archiveVolumeDownloadServiceName:     config.ArchiveVolumeDownloadServiceName,
		archiveVolumeDownloadServiceProtocol: config.ArchiveVolumeDownloadServiceProtocol,
		archiveVolumeDownloadServicePort:     config.ArchiveVolumeDownloadServicePort,
	}
}

func (a *ZipArchiver) Write(ctx context.Context, collectorName, name, namespace, zipFilePath string, writer func(w io.Writer) error) error {
	logger := log.FromContext(ctx).WithName("ZipArchiver")

	destinationPath := fmt.Sprintf("%s.zip", filepath.Join(archivePath, namespace, name))
	zipFile, err := a.openFile(destinationPath)
	if err != nil {
		return err
	}

	zipWriter := a.zipCreator.NewWriter(zipFile)
	defer func() {
		if closeErr := zipWriter.Close(); closeErr != nil {
			logger.Error(closeErr, "failed to close zip writer")
		}
		if closeErr := zipFile.Close(); closeErr != nil {
			logger.Error(closeErr, "failed to close zip file")
		}
	}()

	zipFileWriter, err := zipWriter.Create(zipFilePath)
	if err != nil {
		return fmt.Errorf("failed to create zip writer for file %s: %w", zipFilePath, err)
	}

	err = writer(zipFileWriter)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", zipFilePath, err)
	}

	state, err := a.parseState(name, namespace)
	if err != nil {
		return err
	}
	state.Add(collectorName)
	err = a.writeState(name, namespace, state)
	if err != nil {
		return err
	}

	return nil
}

func (a *ZipArchiver) Read(_ context.Context, name, namespace string) ([]string, error) {
	state, err := a.parseState(name, namespace)
	if err != nil {
		return nil, err
	}

	return state.ExecutedCollectors, nil
}

func (a *ZipArchiver) GetDownloadURL(_ context.Context, name, namespace string) string {
	return fmt.Sprintf("%s://%s.%s.svc.cluster.local:%s/%s/%s.zip", a.archiveVolumeDownloadServiceProtocol, a.archiveVolumeDownloadServiceName, namespace, a.archiveVolumeDownloadServicePort, namespace, name)
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

	file, openErr := a.filesystem.OpenFile(path, os.O_RDWR|os.O_APPEND, 0644)
	if openErr != nil {
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
