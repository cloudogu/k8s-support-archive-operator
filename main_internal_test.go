package main

import (
	"context"
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	v1 "github.com/cloudogu/k8s-support-archive-lib/api/v1"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/adapter/config"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	config2 "sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var testCtx = context.Background()
var testOperatorConfig = &config.OperatorConfig{
	Version:                              nil,
	Namespace:                            "test",
	MetricsServiceName:                   "test",
	MetricsServiceProtocol:               "http",
	MetricsServicePort:                   "8080",
	ArchiveVolumeDownloadServicePort:     "8080",
	ArchiveVolumeDownloadServiceName:     "test",
	ArchiveVolumeDownloadServiceProtocol: "http",
}

func Test_main(t *testing.T) {}
func Test_startOperator(t *testing.T) {
	t.Run("should fail to create Manager", func(t *testing.T) {
		// given
		oldVersion := Version
		Version = "invalid"
		defer func() { Version = oldVersion }()

		flags := flag.NewFlagSet("operator", flag.ContinueOnError)

		// when
		err := startOperator(testCtx, nil, testOperatorConfig, flags, []string{})

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "unable to start manager: must specify Config")
	})
	t.Run("should fail to create controller manager", func(t *testing.T) {
		// given
		t.Setenv("NAMESPACE", "ecosystem")

		oldNewManagerFunc := ctrl.NewManager
		oldGetConfigFunc := ctrl.GetConfigOrDie
		defer func() {
			ctrl.NewManager = oldNewManagerFunc
			ctrl.GetConfigOrDie = oldGetConfigFunc
		}()

		ctrl.NewManager = func(config *rest.Config, options manager.Options) (manager.Manager, error) {
			return nil, assert.AnError
		}
		ctrl.GetConfigOrDie = func() *rest.Config {
			return &rest.Config{}
		}

		flags := flag.NewFlagSet("operator", flag.ContinueOnError)

		// when
		err := startOperator(testCtx, nil, testOperatorConfig, flags, []string{})

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "unable to start manager")
	})
}

func Test_configureManager(t *testing.T) {
	t.Run("should fail to configure Manager", func(t *testing.T) {
		// given

		// when
		err := configureManager(nil, nil, nil, nil, nil)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "unable to configure reconciler: must provide a non-nil Manager")
	})
	t.Run("should fail to configure Manager: SupportArchive kind not registered", func(t *testing.T) {
		// given
		managerMock := newMockControllerManager(t)
		managerMock.EXPECT().GetControllerOptions().Return(config2.Controller{})
		managerMock.EXPECT().GetScheme().Return(&runtime.Scheme{})

		// when
		err := configureManager(managerMock, nil, nil, nil, nil)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "unable to configure reconciler: no kind is registered for the type v1.SupportArchive in scheme")
	})
	t.Run("should fail to configure reconciler", func(t *testing.T) {
		// given
		t.Setenv("NAMESPACE", "ecosystem")
		t.Setenv("STAGE", "development")

		oldNewManagerFunc := ctrl.NewManager
		oldGetConfigFunc := ctrl.GetConfigOrDie
		defer func() {
			ctrl.NewManager = oldNewManagerFunc
			ctrl.GetConfigOrDie = oldGetConfigFunc
		}()

		restConfig := &rest.Config{}
		ctrlManMock := newMockControllerManager(t)
		ctrlManMock.EXPECT().GetControllerOptions().Return(config2.Controller{})
		ctrlManMock.EXPECT().GetScheme().Return(runtime.NewScheme())
		ctrlManMock.EXPECT().GetClient().Return(nil)

		ctrl.NewManager = func(config *rest.Config, options manager.Options) (manager.Manager, error) {
			return ctrlManMock, nil
		}
		ctrl.GetConfigOrDie = func() *rest.Config {
			return restConfig
		}

		flags := flag.NewFlagSet("operator", flag.ContinueOnError)

		// when
		err := startOperator(testCtx, restConfig, testOperatorConfig, flags, []string{})

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "unable to configure manager: unable to configure reconciler")
	})
	t.Run("should fail to add health check to controller manager", func(t *testing.T) {
		// given
		t.Setenv("NAMESPACE", "ecosystem")
		t.Setenv("STAGE", "development")

		oldNewManagerFunc := ctrl.NewManager
		oldGetConfigFunc := ctrl.GetConfigOrDie
		defer func() {
			ctrl.NewManager = oldNewManagerFunc
			ctrl.GetConfigOrDie = oldGetConfigFunc
		}()

		logMock := newMockLogSink(t)
		logMock.EXPECT().Init(mock.Anything).Return()
		logMock.EXPECT().WithValues(mock.Anything, mock.Anything).Return(logMock)
		logMock.EXPECT().WithValues(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(logMock)

		restConfig := &rest.Config{}
		ctrlManMock := newMockControllerManager(t)
		ctrlManMock.EXPECT().GetControllerOptions().Return(config2.Controller{SkipNameValidation: newTrue()})
		ctrlManMock.EXPECT().GetScheme().Return(createScheme(t))
		ctrlManMock.EXPECT().GetLogger().Return(logr.New(logMock))
		ctrlManMock.EXPECT().Add(mock.Anything).Return(nil)
		ctrlManMock.EXPECT().GetCache().Return(nil)
		ctrlManMock.EXPECT().AddHealthzCheck("healthz", mock.Anything).Return(assert.AnError)
		ctrlManMock.EXPECT().GetClient().Return(nil)

		ctrl.NewManager = func(config *rest.Config, options manager.Options) (manager.Manager, error) {
			return ctrlManMock, nil
		}
		ctrl.GetConfigOrDie = func() *rest.Config {
			return restConfig
		}

		flags := flag.NewFlagSet("operator", flag.ContinueOnError)

		// when
		err := startOperator(testCtx, restConfig, testOperatorConfig, flags, []string{})

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "unable to configure manager: unable to add checks to the manager: unable to set up health check")
	})
	t.Run("should fail to add readiness check to controller manager", func(t *testing.T) {
		// given
		t.Setenv("NAMESPACE", "ecosystem")
		t.Setenv("STAGE", "development")

		oldNewManagerFunc := ctrl.NewManager
		oldGetConfigFunc := ctrl.GetConfigOrDie
		defer func() {
			ctrl.NewManager = oldNewManagerFunc
			ctrl.GetConfigOrDie = oldGetConfigFunc
		}()

		logMock := newMockLogSink(t)
		logMock.EXPECT().Init(mock.Anything).Return()
		logMock.EXPECT().WithValues(mock.Anything, mock.Anything).Return(logMock)
		logMock.EXPECT().WithValues(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(logMock)

		restConfig := &rest.Config{}
		ctrlManMock := newMockControllerManager(t)
		ctrlManMock.EXPECT().GetControllerOptions().Return(config2.Controller{SkipNameValidation: newTrue()})
		ctrlManMock.EXPECT().GetScheme().Return(createScheme(t))
		ctrlManMock.EXPECT().GetLogger().Return(logr.New(logMock))
		ctrlManMock.EXPECT().Add(mock.Anything).Return(nil)
		ctrlManMock.EXPECT().GetCache().Return(nil)
		ctrlManMock.EXPECT().AddHealthzCheck("healthz", mock.Anything).Return(nil)
		ctrlManMock.EXPECT().AddReadyzCheck("readyz", mock.Anything).Return(assert.AnError)
		ctrlManMock.EXPECT().GetClient().Return(nil)

		ctrl.NewManager = func(config *rest.Config, options manager.Options) (manager.Manager, error) {
			return ctrlManMock, nil
		}
		ctrl.GetConfigOrDie = func() *rest.Config {
			return restConfig
		}

		flags := flag.NewFlagSet("operator", flag.ContinueOnError)

		// when
		err := startOperator(testCtx, restConfig, testOperatorConfig, flags, []string{})

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "unable to configure manager: unable to add checks to the manager: unable to set up ready check")
	})
	t.Run("should fail to start controller manager", func(t *testing.T) {
		// given
		t.Setenv("NAMESPACE", "ecosystem")
		t.Setenv("STAGE", "development")

		oldNewManagerFunc := ctrl.NewManager
		oldGetConfigFunc := ctrl.GetConfigOrDie
		oldSignalHandlerFunc := ctrl.SetupSignalHandler
		defer func() {
			ctrl.NewManager = oldNewManagerFunc
			ctrl.GetConfigOrDie = oldGetConfigFunc
			ctrl.SetupSignalHandler = oldSignalHandlerFunc
		}()

		logMock := newMockLogSink(t)
		logMock.EXPECT().Init(mock.Anything).Return()
		logMock.EXPECT().WithValues(mock.Anything, mock.Anything).Return(logMock)
		logMock.EXPECT().WithValues(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(logMock)

		restConfig := &rest.Config{}
		ctrlManMock := newMockControllerManager(t)
		ctrlManMock.EXPECT().GetControllerOptions().Return(config2.Controller{SkipNameValidation: newTrue()})
		ctrlManMock.EXPECT().GetScheme().Return(createScheme(t))
		ctrlManMock.EXPECT().GetLogger().Return(logr.New(logMock))
		ctrlManMock.EXPECT().Add(mock.Anything).Return(nil)
		ctrlManMock.EXPECT().GetCache().Return(nil)
		ctrlManMock.EXPECT().AddHealthzCheck("healthz", mock.Anything).Return(nil)
		ctrlManMock.EXPECT().AddReadyzCheck("readyz", mock.Anything).Return(nil)
		ctrlManMock.EXPECT().Start(mock.Anything).Return(assert.AnError)
		ctrlManMock.EXPECT().GetClient().Return(nil)

		ctrl.NewManager = func(config *rest.Config, options manager.Options) (manager.Manager, error) {
			return ctrlManMock, nil
		}
		ctrl.GetConfigOrDie = func() *rest.Config {
			return restConfig
		}
		ctrl.SetupSignalHandler = func() context.Context {
			return testCtx
		}

		flags := flag.NewFlagSet("operator", flag.ContinueOnError)

		// when
		err := startOperator(testCtx, restConfig, testOperatorConfig, flags, []string{})

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "problem running manager")
	})
	t.Run("should succeed to start controller manager", func(t *testing.T) {
		// given
		t.Setenv("NAMESPACE", "ecosystem")
		t.Setenv("STAGE", "development")

		oldNewManagerFunc := ctrl.NewManager
		oldGetConfigFunc := ctrl.GetConfigOrDie
		oldSignalHandlerFunc := ctrl.SetupSignalHandler
		defer func() {
			ctrl.NewManager = oldNewManagerFunc
			ctrl.GetConfigOrDie = oldGetConfigFunc
			ctrl.SetupSignalHandler = oldSignalHandlerFunc
		}()

		logMock := newMockLogSink(t)
		logMock.EXPECT().Init(mock.Anything).Return()
		logMock.EXPECT().WithValues(mock.Anything, mock.Anything).Return(logMock)
		logMock.EXPECT().WithValues(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(logMock)

		restConfig := &rest.Config{}
		ctrlManMock := newMockControllerManager(t)
		ctrlManMock.EXPECT().GetControllerOptions().Return(config2.Controller{SkipNameValidation: newTrue()})
		ctrlManMock.EXPECT().GetScheme().Return(createScheme(t))
		ctrlManMock.EXPECT().GetLogger().Return(logr.New(logMock))
		ctrlManMock.EXPECT().Add(mock.Anything).Return(nil)
		ctrlManMock.EXPECT().GetCache().Return(nil)
		ctrlManMock.EXPECT().AddHealthzCheck("healthz", mock.Anything).Return(nil)
		ctrlManMock.EXPECT().AddReadyzCheck("readyz", mock.Anything).Return(nil)
		ctrlManMock.EXPECT().Start(mock.Anything).Return(nil)
		ctrlManMock.EXPECT().GetClient().Return(nil)

		ctrl.NewManager = func(config *rest.Config, options manager.Options) (manager.Manager, error) {
			return ctrlManMock, nil
		}
		ctrl.GetConfigOrDie = func() *rest.Config {
			return restConfig
		}
		ctrl.SetupSignalHandler = func() context.Context {
			return testCtx
		}

		flags := flag.NewFlagSet("operator", flag.ContinueOnError)

		// when
		err := startOperator(testCtx, restConfig, testOperatorConfig, flags, []string{})

		// then
		require.NoError(t, err)
	})
}

func Test_startK8sManager(t *testing.T) {
	t.Run("should fail to create Manager", func(t *testing.T) {
		// given
		mockManager := newMockControllerManager(t)
		mockManager.EXPECT().Start(testCtx).Return(assert.AnError)

		// when
		err := startK8sManager(testCtx, mockManager)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "problem running manager:")
	})
	t.Run("should succeed", func(t *testing.T) {
		// given
		mockManager := newMockControllerManager(t)
		mockManager.EXPECT().Start(testCtx).Return(nil)

		// when
		err := startK8sManager(testCtx, mockManager)

		// then
		require.NoError(t, err)
	})
}

func Test_addChecks(t *testing.T) {
	t.Run("should fail to set up health check", func(t *testing.T) {
		// given
		mockManager := newMockControllerManager(t)
		mockManager.EXPECT().AddHealthzCheck("healthz", mock.AnythingOfType("healthz.Checker")).Return(assert.AnError)

		// when
		err := addChecks(mockManager)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "unable to set up health check:")
	})
	t.Run("should fail to set up health check", func(t *testing.T) {
		// given
		mockManager := newMockControllerManager(t)
		mockManager.EXPECT().AddHealthzCheck("healthz", mock.AnythingOfType("healthz.Checker")).Return(nil)
		mockManager.EXPECT().AddReadyzCheck("readyz", mock.AnythingOfType("healthz.Checker")).Return(assert.AnError)

		// when
		err := addChecks(mockManager)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "unable to set up ready check:")
	})
	t.Run("should succeed", func(t *testing.T) {
		// given
		mockManager := newMockControllerManager(t)
		mockManager.EXPECT().AddHealthzCheck("healthz", mock.AnythingOfType("healthz.Checker")).Return(nil)
		mockManager.EXPECT().AddReadyzCheck("readyz", mock.AnythingOfType("healthz.Checker")).Return(nil)

		// when
		err := addChecks(mockManager)

		// then
		require.NoError(t, err)
	})
}

func Test_createSupportArchiveClientSet(t *testing.T) {
	t.Run("should succeed to create clientset", func(t *testing.T) {
		// given
		c := &rest.Config{}

		// when
		got, err := createSupportArchiveClientSet(c)

		// then
		require.NotNil(t, got)
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

func newTrue() *bool {
	b := true
	return &b
}
