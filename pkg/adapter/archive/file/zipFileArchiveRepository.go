package file

import (
	"archive/zip"
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/adapter/filesystem"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/config"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"io"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type zipCreator func(w io.Writer) Zipper

func NewZipWriter(w io.Writer) Zipper {
	return zip.NewWriter(w)
}

type ZipFileArchiveRepository struct {
	filesystem                           volumeFs
	archivesPath                         string
	workPath                             string
	zipCreator                           zipCreator
	archiveVolumeDownloadServiceName     string
	archiveVolumeDownloadServicePort     string
	archiveVolumeDownloadServiceProtocol string
}

func NewZipFileArchiveRepository(archivePath, workPath string, zipCreator zipCreator, config *config.OperatorConfig) *ZipFileArchiveRepository {
	return &ZipFileArchiveRepository{
		filesystem:                           filesystem.FileSystem{},
		archivesPath:                         archivePath,
		workPath:                             workPath,
		zipCreator:                           zipCreator,
		archiveVolumeDownloadServiceName:     config.ArchiveVolumeDownloadServiceName,
		archiveVolumeDownloadServicePort:     config.ArchiveVolumeDownloadServicePort,
		archiveVolumeDownloadServiceProtocol: config.ArchiveVolumeDownloadServiceProtocol,
	}
}

func (z *ZipFileArchiveRepository) Create(ctx context.Context, id domain.SupportArchiveID, streams map[domain.CollectorType]domain.Stream) (string, error) {
	logger := log.FromContext(ctx).WithName("ZipFileArchiveRepository.FinishCollection")
	destinationPath := z.getArchivePath(id)

	err := z.filesystem.MkdirAll(filepath.Dir(destinationPath), 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create zip archive directory: %w", err)
	}

	zipFile, err := z.filesystem.OpenFile(destinationPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", destinationPath, err)
	}

	zipWriter := z.zipCreator(zipFile)
	defer func() {
		if closeErr := zipWriter.Close(); closeErr != nil {
			logger.Error(closeErr, "failed to close zip writer")
		}
		if closeErr := zipFile.Close(); closeErr != nil {
			logger.Error(closeErr, "failed to close zip file")
		}
	}()

	for collector, stream := range streams {
		streamErr := z.rangeOverStream(ctx, collector, stream, zipWriter)
		if streamErr != nil {
			return "", streamErr
		}
	}

	return z.getArchiveURL(id), nil
}

func (z *ZipFileArchiveRepository) getArchiveURL(id domain.SupportArchiveID) string {
	return fmt.Sprintf("%s://%s.%s.svc.cluster.local:%s/%s/%s.zip", z.archiveVolumeDownloadServiceProtocol, z.archiveVolumeDownloadServiceName, id.Namespace, z.archiveVolumeDownloadServicePort, id.Namespace, id.Name)
}

func (z *ZipFileArchiveRepository) rangeOverStream(ctx context.Context, collector domain.CollectorType, stream domain.Stream, zipWriter Zipper) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case data, ok := <-stream.Data:
			if ok {
				dataErr := z.copyDataFromStreamToArchive(zipWriter, collector, data.ID, data.BufferedReader)
				if dataErr != nil {
					return fmt.Errorf("error streaming data: %w", dataErr)
				}
			} else {
				return nil
			}
		}
	}
}

func (z *ZipFileArchiveRepository) copyDataFromStreamToArchive(zipper Zipper, collector domain.CollectorType, path string, dataReader *bufio.Reader) error {
	zipFileWriter, err := zipper.Create(filepath.Join(string(collector), path))
	if err != nil {
		return fmt.Errorf("failed to create zip writer for file %s: %w", path, err)
	}

	_, err = z.filesystem.Copy(zipFileWriter, dataReader)
	if err != nil {
		return fmt.Errorf("failed to copy file %s: %w", path, err)
	}

	return nil
}

func (z *ZipFileArchiveRepository) Delete(ctx context.Context, id domain.SupportArchiveID) error {
	logger := log.FromContext(ctx).WithName("FileCleaner").WithValues("SupportArchiveName", id.Name)
	logger.Info("Remove support archive")

	archiveNamespaceDir := filepath.Join(z.archivesPath, id.Namespace)
	archiveFile := fmt.Sprintf("%s.zip", filepath.Join(archiveNamespaceDir, id.Name))

	var multiErr []error
	if err := z.filesystem.Remove(archiveFile); err != nil {
		multiErr = append(multiErr, fmt.Errorf("failed to remove archive %s: %w", archiveFile, err))
	}

	if err := z.removeDirIfEmpty(archiveNamespaceDir); err != nil {
		multiErr = append(multiErr, err)
	}

	return errors.Join(multiErr...)
}

func (z *ZipFileArchiveRepository) removeDirIfEmpty(path string) error {
	dirEntries, err := z.filesystem.ReadDir(path)
	if err != nil {
		return fmt.Errorf("error reading dir %q: %w", path, err)
	}

	if len(dirEntries) != 0 {
		return nil
	}

	err = z.filesystem.Remove(path)
	if err != nil {
		return fmt.Errorf("error removing empty dir %q: %w", path, err)
	}

	return nil
}

func (z *ZipFileArchiveRepository) Exists(_ context.Context, id domain.SupportArchiveID) (bool, error) {
	destinationPath := z.getArchivePath(id)

	_, err := z.filesystem.Stat(destinationPath)

	if err != nil && os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to check if file %s exists: %w", destinationPath, err)
	}
	return true, nil
}

func (z *ZipFileArchiveRepository) getArchivePath(id domain.SupportArchiveID) string {
	return fmt.Sprintf("%s.zip", filepath.Join(z.archivesPath, id.Namespace, id.Name))
}
