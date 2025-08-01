package kubernetes

import (
	"context"
	v1 "github.com/cloudogu/k8s-support-archive-lib/api/v1"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
	"time"
)

var testCtx = context.Background()

const (
	testSupportArchive = "test-support-archive"
	testNamespace      = "test-namespace"
)

func TestNewSupportArchiveOperator(t *testing.T) {
	// override default controller method to retrieve a kube config
	oldGetConfigDelegate := ctrl.GetConfig
	defer func() { ctrl.GetConfig = oldGetConfigDelegate }()
	ctrl.GetConfig = createTestRestConfig

	t.Run("success", func(t *testing.T) {
		// given
		mockClient := newMockSupportArchiveV1Interface(t)

		// when
		doguManager := NewSupportArchiveReconciler(mockClient, newMockCreateArchiveHandler(t), newMockDeleteArchiveHandler(t))

		// then
		require.NotNil(t, doguManager)
	})
}

func createTestRestConfig() (*rest.Config, error) {
	return &rest.Config{}, nil
}

func TestSupportArchiveReconciler_SetupWithManager(t *testing.T) {
	t.Run("should fail", func(t *testing.T) {
		// given
		sut := &SupportArchiveReconciler{}

		// when
		err := sut.SetupWithManager(nil)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "must provide a non-nil Manager")
	})
	t.Run("should succeed", func(t *testing.T) {
		// given
		ctrlManMock := newMockControllerManager(t)
		ctrlManMock.EXPECT().GetControllerOptions().Return(config.Controller{})
		ctrlManMock.EXPECT().GetScheme().Return(createScheme(t))
		logger := log.FromContext(testCtx)
		ctrlManMock.EXPECT().GetLogger().Return(logger)
		ctrlManMock.EXPECT().Add(mock.Anything).Return(nil)
		ctrlManMock.EXPECT().GetCache().Return(nil)

		sut := &SupportArchiveReconciler{}

		// when
		err := sut.SetupWithManager(ctrlManMock)

		// then
		require.NoError(t, err)
	})
}

func createScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	gv, err := schema.ParseGroupVersion("k8s.cloudogu.com/v1")
	assert.NoError(t, err)

	scheme.AddKnownTypes(gv, &v1.SupportArchive{})
	return scheme
}

func TestSupportArchiveReconciler_Reconcile(t *testing.T) {
	archiveCr := &v1.SupportArchive{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSupportArchive,
			Namespace: testNamespace,
		},
	}

	archiveId := domain.SupportArchiveID{
		Namespace: testNamespace,
		Name:      testSupportArchive,
	}

	deletedArchiveCr := &v1.SupportArchive{
		ObjectMeta: metav1.ObjectMeta{
			Name:              testSupportArchive,
			Namespace:         testNamespace,
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
		},
	}

	t.Run("should proceed with archive creation", func(t *testing.T) {
		// given
		request := ctrl.Request{NamespacedName: types.NamespacedName{Name: testSupportArchive, Namespace: testNamespace}}
		mockV1Interface := newMockSupportArchiveV1Interface(t)
		mockInterface := newMockSupportArchiveInterface(t)
		mockV1Interface.EXPECT().SupportArchives(testNamespace).Return(mockInterface)
		mockInterface.EXPECT().Get(testCtx, testSupportArchive, metav1.GetOptions{}).Return(archiveCr, nil)
		archiveHandlerMock := newMockCreateArchiveHandler(t)
		archiveHandlerMock.EXPECT().HandleArchiveRequest(testCtx, archiveCr).Return(true, nil)

		sut := &SupportArchiveReconciler{
			client:        mockV1Interface,
			createHandler: archiveHandlerMock,
		}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{Requeue: true}, actual)
	})

	t.Run("should not requeue if archive handler completed archive creation", func(t *testing.T) {
		// given
		request := ctrl.Request{NamespacedName: types.NamespacedName{Name: testSupportArchive, Namespace: testNamespace}}
		mockV1Interface := newMockSupportArchiveV1Interface(t)
		mockInterface := newMockSupportArchiveInterface(t)
		mockV1Interface.EXPECT().SupportArchives(testNamespace).Return(mockInterface)
		mockInterface.EXPECT().Get(testCtx, testSupportArchive, metav1.GetOptions{}).Return(archiveCr, nil)
		archiveHandlerMock := newMockCreateArchiveHandler(t)
		archiveHandlerMock.EXPECT().HandleArchiveRequest(testCtx, archiveCr).Return(false, nil)

		sut := &SupportArchiveReconciler{
			client:        mockV1Interface,
			createHandler: archiveHandlerMock,
		}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{Requeue: false}, actual)
	})

	t.Run("should cleanup if the request is not found", func(t *testing.T) {
		// given
		request := ctrl.Request{NamespacedName: types.NamespacedName{Name: testSupportArchive, Namespace: testNamespace}}
		mockV1Interface := newMockSupportArchiveV1Interface(t)
		mockInterface := newMockSupportArchiveInterface(t)
		mockV1Interface.EXPECT().SupportArchives(testNamespace).Return(mockInterface)
		mockInterface.EXPECT().Get(testCtx, testSupportArchive, metav1.GetOptions{}).Return(nil, errors.NewNotFound(schema.GroupResource{}, "not found"))
		archiveCleanerMock := newMockDeleteArchiveHandler(t)
		archiveCleanerMock.EXPECT().Delete(testCtx, archiveId).Return(nil)

		sut := &SupportArchiveReconciler{
			client:        mockV1Interface,
			deleteHandler: archiveCleanerMock,
		}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{Requeue: false}, actual)
	})

	t.Run("should cleanup if deletion timestamp is set", func(t *testing.T) {
		// given
		request := ctrl.Request{NamespacedName: types.NamespacedName{Name: testSupportArchive, Namespace: testNamespace}}
		mockV1Interface := newMockSupportArchiveV1Interface(t)
		mockInterface := newMockSupportArchiveInterface(t)
		mockV1Interface.EXPECT().SupportArchives(testNamespace).Return(mockInterface)
		mockInterface.EXPECT().Get(testCtx, testSupportArchive, metav1.GetOptions{}).Return(deletedArchiveCr, nil)
		archiveCleanerMock := newMockDeleteArchiveHandler(t)
		archiveCleanerMock.EXPECT().Delete(testCtx, archiveId).Return(nil)

		sut := &SupportArchiveReconciler{
			client:        mockV1Interface,
			deleteHandler: archiveCleanerMock,
		}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{Requeue: false}, actual)
	})

	t.Run("should requeue and not block on cleanup errors", func(t *testing.T) {
		// given
		request := ctrl.Request{NamespacedName: types.NamespacedName{Name: testSupportArchive, Namespace: testNamespace}}
		mockV1Interface := newMockSupportArchiveV1Interface(t)
		mockInterface := newMockSupportArchiveInterface(t)
		mockV1Interface.EXPECT().SupportArchives(testNamespace).Return(mockInterface)
		mockInterface.EXPECT().Get(testCtx, testSupportArchive, metav1.GetOptions{}).Return(deletedArchiveCr, nil)
		archiveCleanerMock := newMockDeleteArchiveHandler(t)
		archiveCleanerMock.EXPECT().Delete(testCtx, archiveId).Return(assert.AnError)

		sut := &SupportArchiveReconciler{
			client:        mockV1Interface,
			deleteHandler: archiveCleanerMock,
		}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Equal(t, ctrl.Result{}, actual)
	})
}
