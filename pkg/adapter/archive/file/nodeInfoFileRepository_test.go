package file

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNodeInfoRepository_Create(t *testing.T) {
	type fields struct {
		baseFileRepo func(t *testing.T) baseFileRepo
		filesystem   func(t *testing.T) volumeFs
		workPath     func(t *testing.T) string
	}
	type args struct {
		id     domain.SupportArchiveID
		sample *domain.LabeledSample
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantErr    func(t *testing.T, err error)
		assertFile func(t *testing.T, workPath string, args args)
	}{
		{
			name: "should return error when creating directory fails and cleanup is triggered",
			fields: fields{
				workPath: func(t *testing.T) string { return "/data/work" },
				baseFileRepo: func(t *testing.T) baseFileRepo {
					m := newMockBaseFileRepo(t)
					m.EXPECT().Delete(testCtx, testID).Return(nil)
					return m
				},
				filesystem: func(t *testing.T) volumeFs {
					fs := newMockVolumeFs(t)
					dir := filepath.Join("/data/work", testNamespace, testName, archiveNodeInfoDirName)
					fs.EXPECT().MkdirAll(dir, os.FileMode(0755)).Return(assert.AnError)
					return fs
				},
			},
			args: args{
				id: testID,
				sample: &domain.LabeledSample{
					MetricName: "cpu",
					ID:         "node-1",
					Value:      42.5,
					Time:       time.Date(2023, 7, 10, 12, 34, 56, 0, time.FixedZone("UTC", 0)),
				},
			},
			wantErr: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.ErrorContains(t, err, "error creating element from data stream")
				assert.ErrorContains(t, err, "error creating directory for volume node info file")
			},
		},
		{
			name: "should create csv file with header and row and finish collection",
			fields: fields{
				workPath: func(t *testing.T) string { return t.TempDir() },
				baseFileRepo: func(t *testing.T) baseFileRepo {
					m := newMockBaseFileRepo(t)
					m.EXPECT().finishCollection(testCtx, testID).Return(nil)
					return m
				},
				filesystem: func(t *testing.T) volumeFs {
					fs := newMockVolumeFs(t)
					// Allow MkdirAll call on the NodeInfo directory and also create it on the real FS so os.OpenFile succeeds
					return fs
				},
			},
			args: args{
				id: testID,
				sample: &domain.LabeledSample{
					MetricName: "memory",
					ID:         "node-a",
					Value:      7.25,
					Time:       time.Date(2023, 1, 2, 3, 4, 5, 0, time.FixedZone("UTC", 0)),
				},
			},
			wantErr: func(t *testing.T, err error) { require.NoError(t, err) },
			assertFile: func(t *testing.T, workPath string, args args) {
				csvPath := filepath.Join(workPath, testNamespace, testName, archiveNodeInfoDirName, args.sample.MetricName+".csv")
				// Ensure directory exists for MkdirAll; Create will call MkdirAll itself. To be safe, let MkdirAll be a no-op via mock.
				// Read file and verify contents
				data, readErr := os.ReadFile(csvPath)
				require.NoError(t, readErr)
				content := string(data)
				// Header row
				assert.Contains(t, content, "label,value,time\n")
				// Data row
				assert.Contains(t, content, fmt.Sprintf("%s,%.2f,%s\n", args.sample.ID, args.sample.Value, args.sample.Time.Format("2006-01-02T15:04:05-07:00")))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wp := tt.fields.workPath(t)
			fsMock := tt.fields.filesystem(t)
			v := &NodeInfoRepository{
				baseFileRepo:  tt.fields.baseFileRepo(t),
				workPath:      wp,
				filesystem:    fsMock,
				nodeInfoFiles: make(map[metricForID]*os.File),
				writers:       make(map[metricForID]*csv.Writer),
			}

			// For the success case, ensure the directory exists on real FS and set matching expectation on mock
			if tt.assertFile != nil {
				dir := filepath.Join(wp, testNamespace, testName, archiveNodeInfoDirName)
				if m, ok := fsMock.(*mockVolumeFs); ok {
					m.EXPECT().MkdirAll(dir, os.FileMode(0755)).Return(nil)
				}
				_ = os.MkdirAll(dir, os.FileMode(0755))
			}

			dataStream := make(chan *domain.LabeledSample)
			go func() {
				dataStream <- tt.args.sample
				close(dataStream)
			}()
			err := v.Create(testCtx, tt.args.id, dataStream)
			tt.wantErr(t, err)
			if err == nil && tt.assertFile != nil {
				tt.assertFile(t, wp, tt.args)
			}
		})
	}
}
