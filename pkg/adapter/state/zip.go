package state

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"slices"
)

const (
	stateFileFmt = "%s/%s/%s.json"
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

type ZipArchiver struct{}

func NewArchiver() *ZipArchiver {
	return &ZipArchiver{}
}
func (a *ZipArchiver) Write(ctx context.Context, collectorName, name, namespace, zipFilePath string, writer func(w io.Writer) error) error {
	logger := log.FromContext(ctx).WithName("ZipArchiver")

	destinationPath := fmt.Sprintf("%s/%s/%s.zip", archivePath, namespace, name)
	zipFile, err := openFile(destinationPath)
	if err != nil {
		return err
	}

	zipWriter := zip.NewWriter(zipFile)
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

	state, err := parseState(name, namespace)
	if err != nil {
		return err
	}
	state.Add(collectorName)
	err = writeState(name, namespace, state)
	if err != nil {
		return err
	}

	return nil
}

func (a *ZipArchiver) Read(name, namespace string) ([]string, error) {
	state, err := parseState(name, namespace)
	if err != nil {
		return nil, err
	}

	return state.ExecutedCollectors, nil
}

func (a *ZipArchiver) GetDownloadURL(name, namespace string) string {
	return fmt.Sprintf("k8s-support-operator-webserver.%s.svc.cluster.local/%s/%s.zip", namespace, namespace, name)
}

func parseState(name, namespace string) (State, error) {
	path := fmt.Sprintf(stateFileFmt, statePath, namespace, name)
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return State{}, nil
	}

	if err != nil {
		return State{}, fmt.Errorf("failed to stat state file %s: %w", path, err)
	}

	file, err := openFile(path)
	if err != nil {
		return State{}, err
	}

	data, err := io.ReadAll(file)
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

func writeState(name, namespace string, state State) error {
	path := fmt.Sprintf(stateFileFmt, statePath, namespace, name)

	marshal, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	err = createFileIfNotExists(path)
	if err != nil {
		return fmt.Errorf("failed to create state file %s: %w", path, err)
	}

	err = os.WriteFile(path, marshal, 0644)
	if err != nil {
		return fmt.Errorf("failed to write state file %s: %w", path, err)
	}

	return nil
}

func openFile(path string) (*os.File, error) {
	err := createFileIfNotExists(path)
	if err != nil {
		return nil, err
	}

	file, openErr := os.OpenFile(path, os.O_RDWR|os.O_APPEND, 0644)
	if openErr != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", path, err)
	}

	return file, nil
}

func createFileIfNotExists(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		dir := filepath.Dir(path)
		dirErr := os.MkdirAll(dir, 0775)
		if dirErr != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, dirErr)
		}

		_, createErr := os.Create(path)
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

	archiveNamespaceDir := fmt.Sprintf("%s/%s", archivePath, namespace)
	archiveFile := fmt.Sprintf("%s/%s.zip", archiveNamespaceDir, name)

	var multiErr []error
	if err := os.Remove(archiveFile); err != nil {
		multiErr = append(multiErr, err)
	}

	if err := removeDirIfEmpty(archiveNamespaceDir); err != nil {
		multiErr = append(multiErr, err)
	}

	stateNamespaceDir := fmt.Sprintf("%s/%s", statePath, namespace)
	stateFile := fmt.Sprintf("%s/%s.json", stateNamespaceDir, name)
	if err := os.Remove(stateFile); err != nil {
		multiErr = append(multiErr, err)
	}
	if err := removeDirIfEmpty(stateNamespaceDir); err != nil {
		multiErr = append(multiErr, err)
	}

	return errors.Join(multiErr...)
}

func removeDirIfEmpty(path string) error {
	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("error reading dir %q: %w", path, err)
	}

	if len(dirEntries) != 0 {
		return nil
	}

	err = os.Remove(path)
	if err != nil {
		return fmt.Errorf("error removing empty dir %q: %w", path, err)
	}

	return nil
}
