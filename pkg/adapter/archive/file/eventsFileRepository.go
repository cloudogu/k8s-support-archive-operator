package file

const (
	archiveEventsDirName = "Events"
)

type EventFileRepository struct {
	*SingleLogFileRepository
}

func NewEventFileRepository(workPath string, fs volumeFs) *LogFileRepository {
	return &LogFileRepository{
		NewSingleLogFileRepository(workPath, archiveEventsDirName, fs),
	}
}
