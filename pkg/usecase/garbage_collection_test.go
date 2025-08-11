package usecase

import (
	"context"
	"fmt"
	libv1 "github.com/cloudogu/k8s-support-archive-lib/api/v1"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewGarbageCollectionUseCase(t *testing.T) {
	result := NewGarbageCollectionUseCase(newMockSupportArchiveInterface(t), newMockSupportArchiveRepository(t), newMockDeleteArchiveHandler(t), time.Minute, 5)
	assert.NotEmpty(t, result)
	assert.NotEmpty(t, result.supportArchivesInterface)
	assert.NotEmpty(t, result.supportArchiveRepository)
	assert.Equal(t, time.Minute, result.interval)
	assert.Equal(t, 5, result.numberToKeep)
}

func TestGarbageCollectionUseCase_CollectGarbageWithInterval(t *testing.T) {
	type fields struct {
		supportArchivesInterface    func(t *testing.T) supportArchiveInterface
		supportArchiveRepository    func(t *testing.T) supportArchiveRepository
		supportArchiveDeleteHandler func(t *testing.T) deleteArchiveHandler
		interval                    time.Duration
		numberToKeep                int
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should disable garbage collection if interval is set to zero",
			fields: fields{
				supportArchivesInterface: func(t *testing.T) supportArchiveInterface {
					m := newMockSupportArchiveInterface(t)
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
				interval:     0,
				numberToKeep: 5,
			},
			wantErr: assert.NoError,
		},
		{
			name: "should fail to list archives",
			fields: fields{
				supportArchivesInterface: func(t *testing.T) supportArchiveInterface {
					m := newMockSupportArchiveInterface(t)
					m.EXPECT().List(mock.Anything, metav1.ListOptions{}).Return(nil, assert.AnError)
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
				interval:     time.Millisecond,
				numberToKeep: 5,
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to list support archives", i)
			},
		},
		{
			name: "should fail to check if archive exists",
			fields: fields{
				supportArchivesInterface: func(t *testing.T) supportArchiveInterface {
					m := newMockSupportArchiveInterface(t)
					m.EXPECT().List(mock.Anything, metav1.ListOptions{}).
						Return(&libv1.SupportArchiveList{Items: createTestArchiveDescriptors(0, 10)}, nil)
					return m
				},
				supportArchiveRepository: func(t *testing.T) supportArchiveRepository {
					m := newMockSupportArchiveRepository(t)
					for _, archive := range createTestArchiveIDs(0, 10) {
						m.EXPECT().Exists(mock.Anything, archive).Return(false, assert.AnError)
					}
					return m
				},
				supportArchiveDeleteHandler: func(t *testing.T) deleteArchiveHandler {
					m := newMockDeleteArchiveHandler(t)
					return m
				},
				interval:     time.Millisecond,
				numberToKeep: 5,
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					createErrContainsForIDs(t, err, createTestArchiveIDs(0, 10), "failed to check if support archive test-namespace/%s exists", i)
			},
		},
		{
			name: "should fail to delete archive resources",
			fields: fields{
				supportArchivesInterface: func(t *testing.T) supportArchiveInterface {
					m := newMockSupportArchiveInterface(t)
					descriptors := createTestArchiveDescriptors(0, 10)
					m.EXPECT().List(mock.Anything, metav1.ListOptions{}).
						Return(&libv1.SupportArchiveList{Items: descriptors}, nil)
					for _, descriptor := range descriptors[0:5] {
						m.EXPECT().Delete(mock.Anything, descriptor.Name, metav1.DeleteOptions{}).Return(assert.AnError)
					}
					return m
				},
				supportArchiveRepository: func(t *testing.T) supportArchiveRepository {
					m := newMockSupportArchiveRepository(t)
					for _, archive := range createTestArchiveIDs(0, 10) {
						m.EXPECT().Exists(mock.Anything, archive).Return(true, nil)
					}
					return m
				},
				supportArchiveDeleteHandler: func(t *testing.T) deleteArchiveHandler {
					m := newMockDeleteArchiveHandler(t)
					return m
				},
				interval:     time.Millisecond,
				numberToKeep: 5,
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					createErrContainsForIDs(t, err, createTestArchiveIDs(0, 5), "failed to delete support archive resource test-namespace/%s", i)
			},
		},
		{
			name: "should fail to delete stored archives",
			fields: fields{
				supportArchivesInterface: func(t *testing.T) supportArchiveInterface {
					m := newMockSupportArchiveInterface(t)
					descriptors := createTestArchiveDescriptors(0, 10)
					m.EXPECT().List(mock.Anything, metav1.ListOptions{}).
						Return(&libv1.SupportArchiveList{Items: descriptors}, nil)
					for _, descriptor := range descriptors[:5] {
						m.EXPECT().Delete(mock.Anything, descriptor.Name, metav1.DeleteOptions{}).Return(nil)
					}
					return m
				},
				supportArchiveRepository: func(t *testing.T) supportArchiveRepository {
					m := newMockSupportArchiveRepository(t)
					for _, archive := range createTestArchiveIDs(0, 10) {
						m.EXPECT().Exists(mock.Anything, archive).Return(true, nil)
					}
					return m
				},
				supportArchiveDeleteHandler: func(t *testing.T) deleteArchiveHandler {
					m := newMockDeleteArchiveHandler(t)
					archives := createTestArchiveIDs(0, 10)
					for _, archive := range archives[:5] {
						m.EXPECT().Delete(mock.Anything, archive).Return(assert.AnError)
					}
					return m
				},
				interval:     time.Millisecond,
				numberToKeep: 5,
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					createErrContainsForIDs(t, err, createTestArchiveIDs(0, 5), "failed to delete stored support archive test-namespace/%s", i)
			},
		},
		{
			name: "should succeed to delete archives that exist and are not to keep",
			fields: fields{
				supportArchivesInterface: func(t *testing.T) supportArchiveInterface {
					m := newMockSupportArchiveInterface(t)
					descriptors := createTestArchiveDescriptors(0, 10)
					m.EXPECT().List(mock.Anything, metav1.ListOptions{}).
						Return(&libv1.SupportArchiveList{Items: descriptors}, nil)
					for _, descriptor := range descriptors[3:5] {
						m.EXPECT().Delete(mock.Anything, descriptor.Name, metav1.DeleteOptions{}).Return(nil)
					}
					return m
				},
				supportArchiveRepository: func(t *testing.T) supportArchiveRepository {
					m := newMockSupportArchiveRepository(t)
					for i, archive := range createTestArchiveIDs(0, 10) {
						m.EXPECT().Exists(mock.Anything, archive).Return(i > 2, nil)
					}
					return m
				},
				supportArchiveDeleteHandler: func(t *testing.T) deleteArchiveHandler {
					m := newMockDeleteArchiveHandler(t)
					archives := createTestArchiveIDs(0, 10)
					for _, archive := range archives[3:5] {
						m.EXPECT().Delete(mock.Anything, archive).Return(nil)
					}
					return m
				},
				interval:     time.Millisecond,
				numberToKeep: 5,
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &GarbageCollectionUseCase{
				supportArchivesInterface:    tt.fields.supportArchivesInterface(t),
				supportArchiveRepository:    tt.fields.supportArchiveRepository(t),
				supportArchiveDeleteHandler: tt.fields.supportArchiveDeleteHandler(t),
				interval:                    tt.fields.interval,
				numberToKeep:                tt.fields.numberToKeep,
			}
			ctx, cancel := context.WithTimeout(testCtx, 5*time.Millisecond)
			defer cancel()

			tt.wantErr(t, g.CollectGarbageWithInterval(ctx), fmt.Sprintf("CollectGarbageWithInterval(%v)", ctx))
		})
	}
}
