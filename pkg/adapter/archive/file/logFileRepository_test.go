package file

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewLogFileRepository(t *testing.T) {
	// given
	fsMock := newMockVolumeFs(t)

	// when
	repository := NewLogFileRepository(testWorkPath, fsMock)

	// then
	assert.NotNil(t, repository.files)
	assert.Equal(t, fsMock, repository.filesystem)
	assert.Equal(t, testWorkPath, repository.workPath)
	assert.Equal(t, "Logs", repository.dirName)
}
