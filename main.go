package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	// Import all Kubernetes client auth plugins (e.g., Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	k8scloudogucomv1 "github.com/cloudogu/k8s-support-archive-lib/api/v1"
	"github.com/cloudogu/k8s-support-archive-lib/client"
	k8scloudoguclient "github.com/cloudogu/k8s-support-archive-lib/client"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	config2 "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/adapter/archive/file"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/adapter/collector"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/adapter/config"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/adapter/filesystem"
	adapterK8s "github.com/cloudogu/k8s-support-archive-operator/pkg/adapter/kubernetes"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/adapter/prometheus"
	v1 "github.com/cloudogu/k8s-support-archive-operator/pkg/adapter/prometheus/v1"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/usecase"
)

var (
	// Version of the application
	Version = "0.0.0"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

const (
	archivePath = "/data/support-archives"
	workPath    = "/data/work"
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(k8scloudogucomv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

type ecosystemClientSet struct {
	kubernetes.Interface
	client.SupportArchiveEcosystemInterface
}

// nolint:gocyclo
func main() {
	ctx := ctrl.SetupSignalHandler()

	config.ConfigureLogger()

	restConfig := config2.GetConfigOrDie()
	operatorConfig, err := config.NewOperatorConfig(Version)
	if err != nil {
		setupLog.Error(err, "unable to create operator config")
		os.Exit(1)
	}
	err = startOperator(ctx, restConfig, operatorConfig, flag.CommandLine, os.Args)
	if err != nil {
		setupLog.Error(err, "unable to start operator")
		os.Exit(1)
	}
}

func startOperator(
	ctx context.Context,
	restConfig *rest.Config,
	operatorConfig *config.OperatorConfig,
	flags *flag.FlagSet,
	args []string,
) error {
	k8sManager, err := NewK8sManager(restConfig, operatorConfig, flags, args)
	if err != nil {
		return fmt.Errorf("unable to start manager: %w", err)
	}

	supportArchiveClient, err := createSupportArchiveClientSet(restConfig)
	if err != nil {
		return fmt.Errorf("unable to create client set: %w", err)
	}

	k8sClientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("unable to create kubernetes client set: %w", err)
	}

	ecoClientSet := ecosystemClientSet{
		k8sClientSet,
		supportArchiveClient,
	}

	v1SupportArchive := ecoClientSet.SupportArchiveV1()
	supportArchiveRepository := file.NewZipFileArchiveRepository(archivePath, file.NewZipWriter, operatorConfig)

	fs := filesystem.FileSystem{}

	logCollector := collector.NewLogCollector()
	logRepository := file.NewLogFileRepository(workPath, fs)

	address := fmt.Sprintf("%s://%s.%s.svc.cluster.local:%s", operatorConfig.MetricsServiceProtocol, operatorConfig.MetricsServiceName, operatorConfig.Namespace, operatorConfig.MetricsServicePort)
	// TODO Implement ServiceAccount for Prometheus. Create secret in Prometheus Chart and use it?
	metricsClient, err := prometheus.GetClient(address, "")
	if err != nil {
		return fmt.Errorf("unable to create prometheus client: %w", err)
	}
	metricsCollector := v1.NewPrometheusMetricsV1API(metricsClient)

	volumesCollector := collector.NewVolumesCollector(ecoClientSet.CoreV1(), metricsCollector)
	volumeRepository := file.NewVolumesFileRepository(workPath, fs)

	mapping := make(map[domain.CollectorType]usecase.CollectorAndRepository)
	mapping[domain.CollectorTypeLog] = usecase.CollectorAndRepository{Collector: logCollector, Repository: logRepository}
	mapping[domain.CollectorTypVolumeInfo] = usecase.CollectorAndRepository{Collector: volumesCollector, Repository: volumeRepository}

	createUseCase := usecase.NewCreateArchiveUseCase(v1SupportArchive, mapping, supportArchiveRepository)
	deleteUseCase := usecase.NewDeleteArchiveUseCase(mapping, supportArchiveRepository)
	r := adapterK8s.NewSupportArchiveReconciler(v1SupportArchive, createUseCase, deleteUseCase)

	reconciliationTrigger := make(chan event.GenericEvent)
	syncHandler := usecase.NewSyncArchiveUseCase(v1SupportArchive, supportArchiveRepository, deleteUseCase, operatorConfig.SupportArchiveSyncInterval, operatorConfig.Namespace, reconciliationTrigger)

	garbageCollectionHandler := usecase.NewGarbageCollectionUseCase(v1SupportArchive.SupportArchives(operatorConfig.Namespace), supportArchiveRepository, operatorConfig.GarbageCollectionInterval, operatorConfig.GarbageCollectionNumberToKeep)
	err = configureManager(k8sManager, r, reconciliationTrigger, syncHandler, garbageCollectionHandler)
	if err != nil {
		return fmt.Errorf("unable to configure manager: %w", err)
	}

	err = startK8sManager(ctx, k8sManager)
	if err != nil {
		return fmt.Errorf("unable to start operator: %w", err)
	}
	return err
}

func NewK8sManager(
	restConfig *rest.Config,
	operatorConfig *config.OperatorConfig,
	flags *flag.FlagSet, args []string,
) (manager.Manager, error) {
	options := getK8sManagerOptions(flags, args, operatorConfig)
	return ctrl.NewManager(restConfig, options)
}

func configureManager(k8sManager controllerManager, supportArchiveReconciler *adapterK8s.SupportArchiveReconciler, trigger chan event.GenericEvent, syncHandler *usecase.SyncArchiveUseCase, garbageCollectionHandler *usecase.GarbageCollectionUseCase) error {
	err := supportArchiveReconciler.SetupWithManager(k8sManager, trigger)
	if err != nil {
		return fmt.Errorf("unable to configure reconciler: %w", err)
	}

	err = k8sManager.Add(manager.RunnableFunc(func(ctx context.Context) error {
		return syncHandler.SyncArchivesWithInterval(ctx)
	}))
	if err != nil {
		return fmt.Errorf("unable to add sync archive handler: %w", err)
	}

	err = k8sManager.Add(manager.RunnableFunc(func(ctx context.Context) error {
		return garbageCollectionHandler.CollectGarbageWithInterval(ctx)
	}))
	if err != nil {
		return fmt.Errorf("unable to add garbage collection handler: %w", err)
	}

	err = addChecks(k8sManager)
	if err != nil {
		return fmt.Errorf("unable to add checks to the manager: %w", err)
	}

	return nil
}

func getK8sManagerOptions(flags *flag.FlagSet, args []string, operatorConfig *config.OperatorConfig) ctrl.Options {
	controllerOpts := ctrl.Options{
		Scheme: scheme,
		Cache: cache.Options{DefaultNamespaces: map[string]cache.Config{
			operatorConfig.Namespace: {},
		}},
	}
	controllerOpts = parseManagerFlags(flags, args, controllerOpts)

	return controllerOpts
}

func parseManagerFlags(flags *flag.FlagSet, args []string, ctrlOpts ctrl.Options) ctrl.Options {
	var metricsAddr string
	var probeAddr string
	flags.StringVar(&metricsAddr, "metrics-bind-address", ":8082", "The address the metric endpoint binds to.")
	flags.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")

	// Ignore errors; flags is set to exit on errors
	_ = flags.Parse(args)

	ctrlOpts.Metrics = metricsserver.Options{BindAddress: metricsAddr}
	ctrlOpts.HealthProbeBindAddress = probeAddr

	return ctrlOpts
}

func addChecks(k8sManager controllerManager) error {
	if err := k8sManager.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up health check: %w", err)
	}
	if err := k8sManager.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up ready check: %w", err)
	}

	return nil
}

func startK8sManager(ctx context.Context, k8sManager controllerManager) error {
	logger := log.FromContext(ctx).WithName("k8s-manager-start")
	logger.Info("starting manager")
	if err := k8sManager.Start(ctx); err != nil {
		return fmt.Errorf("problem running manager: %w", err)
	}

	return nil
}

func createSupportArchiveClientSet(restConfig *rest.Config) (k8scloudoguclient.SupportArchiveEcosystemInterface, error) {
	supportArchiveClientSet, err := k8scloudoguclient.NewSupportArchiveClientSet(restConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create ecosystem clientset: %w", err)
	}
	return supportArchiveClientSet, nil
}
