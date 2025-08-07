package usecase

import (
	"context"
	"fmt"
	libv1 "github.com/cloudogu/k8s-support-archive-lib/api/v1"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"testing"
	"time"
)

func TestNewSyncArchiveUseCase(t *testing.T) {
	sut := NewSyncArchiveUseCase(newMockSupportArchiveV1Interface(t), newMockSupportArchiveRepository(t), newMockDeleteArchiveHandler(t), time.Minute, testArchiveNamespace, make(chan<- event.GenericEvent))
	assert.NotEmpty(t, sut)
	assert.NotNil(t, sut.supportArchivesInterface)
	assert.NotNil(t, sut.supportArchiveRepository)
	assert.NotNil(t, sut.supportArchiveDeleteHandler)
	assert.NotNil(t, sut.syncInterval)
	assert.NotNil(t, sut.namespace)
	assert.NotNil(t, sut.reconciliationTrigger)
}

func TestSyncArchiveUseCase_SyncArchivesWithInterval(t *testing.T) {
	type fields struct {
		supportArchivesInterface    func(t *testing.T) supportArchiveV1Interface
		supportArchiveRepository    func(t *testing.T) supportArchiveRepository
		supportArchiveDeleteHandler func(t *testing.T) deleteArchiveHandler
		syncInterval                time.Duration
	}
	tests := []struct {
		name       string
		fields     fields
		wantErr    assert.ErrorAssertionFunc
		wantEvents []event.GenericEvent
	}{
		{
			name: "should disable sync if interval is set to 0",
			fields: fields{
				supportArchivesInterface: func(t *testing.T) supportArchiveV1Interface {
					m := newMockSupportArchiveV1Interface(t)
					return m
				},
				supportArchiveRepository: func(t *testing.T) supportArchiveRepository {
					m := newMockSupportArchiveRepository(t)
					return m
				},
				supportArchiveDeleteHandler: func(t *testing.T) deleteArchiveHandler {
					m := newMockDeleteArchiveHandler(t)
					return m
				},
				syncInterval: time.Duration(0),
			},
			wantErr:    assert.NoError,
			wantEvents: make([]event.GenericEvent, 0),
		},
		{
			name: "should fail to list stored support archives",
			fields: fields{
				supportArchivesInterface: func(t *testing.T) supportArchiveV1Interface {
					m := newMockSupportArchiveV1Interface(t)
					return m
				},
				supportArchiveRepository: func(t *testing.T) supportArchiveRepository {
					m := newMockSupportArchiveRepository(t)
					m.EXPECT().List(mock.Anything).Return(nil, assert.AnError)
					return m
				},
				supportArchiveDeleteHandler: func(t *testing.T) deleteArchiveHandler {
					m := newMockDeleteArchiveHandler(t)
					return m
				},
				syncInterval: time.Millisecond,
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to list stored support archives", i)
			},
			wantEvents: make([]event.GenericEvent, 0),
		},
		{
			name: "should fail to list support archive descriptors",
			fields: fields{
				supportArchivesInterface: func(t *testing.T) supportArchiveV1Interface {
					sam := newMockSupportArchiveInterface(t)
					sam.EXPECT().List(mock.Anything, metav1.ListOptions{}).Return(nil, assert.AnError)
					m := newMockSupportArchiveV1Interface(t)
					m.EXPECT().SupportArchives(testArchiveNamespace).Return(sam)
					return m
				},
				supportArchiveRepository: func(t *testing.T) supportArchiveRepository {
					m := newMockSupportArchiveRepository(t)
					m.EXPECT().List(mock.Anything).Return(nil, nil)
					return m
				},
				supportArchiveDeleteHandler: func(t *testing.T) deleteArchiveHandler {
					m := newMockDeleteArchiveHandler(t)
					return m
				},
				syncInterval: time.Millisecond,
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to list support archive descriptors", i)
			},
			wantEvents: make([]event.GenericEvent, 0),
		},
		{
			name: "should fail to delete stored support archives",
			fields: fields{
				supportArchivesInterface: func(t *testing.T) supportArchiveV1Interface {
					sam := newMockSupportArchiveInterface(t)
					sam.EXPECT().List(mock.Anything, metav1.ListOptions{}).Return(&libv1.SupportArchiveList{}, nil)
					m := newMockSupportArchiveV1Interface(t)
					m.EXPECT().SupportArchives(testArchiveNamespace).Return(sam)
					return m
				},
				supportArchiveRepository: func(t *testing.T) supportArchiveRepository {
					m := newMockSupportArchiveRepository(t)
					archives := createTestArchiveIDs(0, 2)
					m.EXPECT().List(mock.Anything).Return(archives, nil)
					return m
				},
				supportArchiveDeleteHandler: func(t *testing.T) deleteArchiveHandler {
					m := newMockDeleteArchiveHandler(t)
					archives := createTestArchiveIDs(0, 2)
					for _, archive := range archives {
						m.EXPECT().Delete(mock.Anything, archive).Return(assert.AnError)
					}
					return m
				},
				syncInterval: time.Millisecond,
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					createErrContainsForIDs(t, err, createTestArchiveIDs(0, 2), "failed to delete support archive %q", i)
			},
			wantEvents: make([]event.GenericEvent, 0),
		},
		{
			name: "should succeed",
			fields: fields{
				supportArchivesInterface: func(t *testing.T) supportArchiveV1Interface {
					sam := newMockSupportArchiveInterface(t)
					archives := createTestArchiveDescriptors(2, 6)
					sam.EXPECT().List(mock.Anything, metav1.ListOptions{}).Return(&libv1.SupportArchiveList{Items: archives}, nil)
					m := newMockSupportArchiveV1Interface(t)
					m.EXPECT().SupportArchives(testArchiveNamespace).Return(sam)
					return m
				},
				supportArchiveRepository: func(t *testing.T) supportArchiveRepository {
					m := newMockSupportArchiveRepository(t)
					archives := createTestArchiveIDs(0, 4)
					m.EXPECT().List(mock.Anything).Return(archives, nil)
					return m
				},
				supportArchiveDeleteHandler: func(t *testing.T) deleteArchiveHandler {
					m := newMockDeleteArchiveHandler(t)
					archives := createTestArchiveIDs(0, 2)
					for _, archive := range archives {
						m.EXPECT().Delete(mock.Anything, archive).Return(nil)
					}
					return m
				},
				syncInterval: time.Millisecond,
			},
			wantErr:    assert.NoError,
			wantEvents: createTestEvents(createTestArchiveDescriptors(2, 6)),
		},
		{
			name: "should succeed with partial failure when listing stored support archives",
			fields: fields{
				supportArchivesInterface: func(t *testing.T) supportArchiveV1Interface {
					sam := newMockSupportArchiveInterface(t)
					archives := createTestArchiveDescriptors(2, 6)
					sam.EXPECT().List(mock.Anything, metav1.ListOptions{}).Return(&libv1.SupportArchiveList{Items: archives}, nil)
					m := newMockSupportArchiveV1Interface(t)
					m.EXPECT().SupportArchives(testArchiveNamespace).Return(sam)
					return m
				},
				supportArchiveRepository: func(t *testing.T) supportArchiveRepository {
					m := newMockSupportArchiveRepository(t)
					archives := createTestArchiveIDs(0, 4)
					m.EXPECT().List(mock.Anything).Return(archives, assert.AnError)
					return m
				},
				supportArchiveDeleteHandler: func(t *testing.T) deleteArchiveHandler {
					m := newMockDeleteArchiveHandler(t)
					archives := createTestArchiveIDs(0, 2)
					for _, archive := range archives {
						m.EXPECT().Delete(mock.Anything, archive).Return(nil)
					}
					return m
				},
				syncInterval: time.Millisecond,
			},
			wantErr:    assert.NoError,
			wantEvents: createTestEvents(createTestArchiveDescriptors(2, 6)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reconciliationTrigger := make(chan event.GenericEvent)
			s := &SyncArchiveUseCase{
				supportArchivesInterface:    tt.fields.supportArchivesInterface(t),
				supportArchiveRepository:    tt.fields.supportArchiveRepository(t),
				supportArchiveDeleteHandler: tt.fields.supportArchiveDeleteHandler(t),
				namespace:                   testArchiveNamespace,
				syncInterval:                tt.fields.syncInterval,
				reconciliationTrigger:       reconciliationTrigger,
			}

			ctx, cancel := context.WithCancel(testCtx)
			defer cancel()
			// separate context because otherwise we will stop receiving events before they are sent and it will block indefinitely
			timoutCtx, cancelTimeout := context.WithTimeout(ctx, tt.fields.syncInterval*5)
			defer cancelTimeout()

			// must be in goroutine because otherwise sending events will block
			go func() {
				var receivedEvents []event.GenericEvent
			loop:
				for {
					select {
					case <-ctx.Done():
						break loop
					case receivedEvent := <-reconciliationTrigger:
						receivedEvents = append(receivedEvents, receivedEvent)
					}
				}
				assert.Subset(t, receivedEvents, tt.wantEvents)
			}()

			tt.wantErr(t, s.SyncArchivesWithInterval(timoutCtx), fmt.Sprintf("SyncArchivesWithInterval(%v)", ctx))
		})
	}
}

func createErrContainsForIDs(t assert.TestingT, err error, ids []domain.SupportArchiveID, format string, msgAndArgs ...interface{}) bool {
	for _, id := range ids {
		if assert.ErrorContains(t, err, fmt.Sprintf(format, id.Name), msgAndArgs) == false {
			return false
		}
	}
	return true
}

func createTestArchiveIDs(from, to int) []domain.SupportArchiveID {
	var archives []domain.SupportArchiveID
	for i := from; i < to; i++ {
		archives = append(archives, domain.SupportArchiveID{
			Namespace: testArchiveNamespace,
			Name:      fmt.Sprintf("%s-%d", testArchiveName, i),
		})
	}
	return archives
}

func createTestEvents(archives []libv1.SupportArchive) []event.GenericEvent {
	var events []event.GenericEvent
	for _, archive := range archives {
		events = append(events, event.GenericEvent{Object: &archive})
	}
	return events
}

func createTestArchiveDescriptors(from, to int) []libv1.SupportArchive {
	var archives []libv1.SupportArchive
	for i := from; i < to; i++ {
		archives = append(archives, libv1.SupportArchive{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         testArchiveNamespace,
				Name:              fmt.Sprintf("%s-%d", testArchiveName, i),
				CreationTimestamp: metav1.NewTime(time.Unix(1754490135, int64(-i))),
			},
		})
	}
	return archives
}
