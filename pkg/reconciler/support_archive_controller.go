package reconciler

import (
	"context"
	"errors"
	k8sv1 "github.com/cloudogu/k8s-support-archive-lib/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type SupportArchiveReconciler struct {
	Client EcosystemClientSet
	Scheme *runtime.Scheme
}

func NewSupportArchiveReconciler(client EcosystemClientSet, scheme *runtime.Scheme) *SupportArchiveReconciler {
	return &SupportArchiveReconciler{
		Client: client,
		Scheme: scheme,
	}
}

func (s *SupportArchiveReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Reconciler is triggered")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (s *SupportArchiveReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if mgr == nil {
		return errors.New("must provide a non-nil Manager")
	}

	controllerOptions := mgr.GetControllerOptions()
	options := controller.TypedOptions[reconcile.Request]{
		SkipNameValidation: controllerOptions.SkipNameValidation,
		RecoverPanic:       controllerOptions.RecoverPanic,
		NeedLeaderElection: controllerOptions.NeedLeaderElection,
	}
	return ctrl.NewControllerManagedBy(mgr).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		WithOptions(options).
		For(&k8sv1.SupportArchive{}).
		Complete(s)
}
