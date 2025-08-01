package file

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/adapter/filesystem"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/config"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

type zipCreator func(w io.Writer) Zipper

func NewZipWriter(w io.Writer) Zipper {
	return zip.NewWriter(w)
}

type ZipFileArchiveRepository struct {
	filesystem                           volumeFs
	zipCreator                           zipCreator
	archivesPath                         string
	archiveVolumeDownloadServiceName     string
	archiveVolumeDownloadServicePort     string
	archiveVolumeDownloadServiceProtocol string
}

func NewZipFileArchiveRepository(archivesPath string, zipCreator zipCreator, config *config.OperatorConfig) *ZipFileArchiveRepository {
	return &ZipFileArchiveRepository{
		filesystem:                           filesystem.FileSystem{},
		archivesPath:                         archivesPath,
		zipCreator:                           zipCreator,
		archiveVolumeDownloadServiceName:     config.ArchiveVolumeDownloadServiceName,
		archiveVolumeDownloadServicePort:     config.ArchiveVolumeDownloadServicePort,
		archiveVolumeDownloadServiceProtocol: config.ArchiveVolumeDownloadServiceProtocol,
	}
}

func (z *ZipFileArchiveRepository) Create(ctx context.Context, id domain.SupportArchiveID, streams map[domain.CollectorType]*domain.Stream) (string, error) {
	logger := log.FromContext(ctx).WithName("ZipFileArchiveRepository.FinishCollection")
	destinationPath := z.getArchivePath(id)

	err := z.filesystem.MkdirAll(filepath.Dir(destinationPath), 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create zip archive directory: %w", err)
	}

	zipFile, err := z.filesystem.OpenFile(destinationPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", destinationPath, err)
	}
	defer func() {
		// If any error occurs during creating, try to delete the zip to avoid false recognition bei Exists method.
		if err == nil {
			return
		}
		err = z.filesystem.Remove(destinationPath)
		if err != nil {
			logger.Error(err, fmt.Sprintf("failed to remove zip file %s after error: %s", destinationPath, err))
		}
	}()

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
		err = z.rangeOverStream(ctx, collector, stream, zipWriter)
		if err != nil {
			return "", err
		}
	}

	return z.getArchiveURL(id), nil
}

func (z *ZipFileArchiveRepository) getArchiveURL(id domain.SupportArchiveID) string {
	return fmt.Sprintf("%s://%s.%s.svc.cluster.local:%s/%s/%s.zip", z.archiveVolumeDownloadServiceProtocol, z.archiveVolumeDownloadServiceName, id.Namespace, z.archiveVolumeDownloadServicePort, id.Namespace, id.Name)
}

func (z *ZipFileArchiveRepository) rangeOverStream(ctx context.Context, collector domain.CollectorType, stream *domain.Stream, zipWriter Zipper) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case data, ok := <-stream.Data:
			if ok {
				reader, closeReader, err := data.StreamConstructor()
				if err != nil {
					return fmt.Errorf("failed to construct reader for file %s: %w", data.ID, err)
				}

				dataErr := z.copyDataFromStreamToArchive(zipWriter, collector, data.ID, reader)
				if dataErr != nil {
					return fmt.Errorf("error streaming data: %w", dataErr)
				}

				err = closeReader()
				if err != nil {
					return fmt.Errorf("error closing reader for file %s: %w", data.ID, err)
				}
			} else {
				return nil
			}
		}
	}
}

func (z *ZipFileArchiveRepository) copyDataFromStreamToArchive(zipper Zipper, collector domain.CollectorType, path string, dataReader io.Reader) error {
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

func (z *ZipFileArchiveRepository) List(_ context.Context) ([]domain.SupportArchiveID, error) {
	archiveMatcher := regexp.MustCompile(fmt.Sprintf("%s/%s", regexp.QuoteMeta(z.archivesPath), `(?P<namespace>[^/]+)/(?P<name>[^/.]+)\.zip`))
	namespaceIndex := archiveMatcher.SubexpIndex("namespace")
	nameIndex := archiveMatcher.SubexpIndex("name")

	var list []domain.SupportArchiveID
	err := z.filesystem.WalkDir(z.archivesPath, func(path string, d fs.DirEntry, err error) error {
		errs := []error{err}
		if !d.IsDir() {
			matches := archiveMatcher.FindStringSubmatch(path)
			if matches == nil || len(matches) != 3 {
				errs = append(errs, fmt.Errorf("failed to match path %q: not an archive", path))
			} else {
				list = append(list, domain.SupportArchiveID{
					Namespace: matches[namespaceIndex],
					Name:      matches[nameIndex],
				})
			}
		}

		return errors.Join(errs...)
	})

	return list, err
}

func (z *ZipFileArchiveRepository) getArchivePath(id domain.SupportArchiveID) string {
	return fmt.Sprintf("%s.zip", filepath.Join(z.archivesPath, id.Namespace, id.Name))
}
