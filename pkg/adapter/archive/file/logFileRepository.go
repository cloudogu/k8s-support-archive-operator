package file

const (
	archiveLogDirName = "Logs"
)

type LogFileRepository struct {
	*SingleLogFileRepository
}

func NewLogFileRepository(workPath string, fs volumeFs) *LogFileRepository {
	return &LogFileRepository{
		NewSingleLogFileRepository(workPath, archiveLogDirName, fs),
	}
}
