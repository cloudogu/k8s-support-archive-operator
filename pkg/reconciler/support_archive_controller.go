package reconciler

import (
	"context"
	"errors"
	"fmt"
	libv1 "github.com/cloudogu/k8s-support-archive-lib/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"slices"
)

const (
	finalizerName = "k8s.cloudogu.com/support-archive-reconciler"
)

type SupportArchiveReconciler struct {
	client         supportArchiveV1Interface
	archiveHandler archiveHandler
	cleaner        archiveCleaner
}

func NewSupportArchiveReconciler(client supportArchiveV1Interface, archiveHandler archiveHandler, cleaner archiveCleaner) *SupportArchiveReconciler {
	return &SupportArchiveReconciler{
		client:         client,
		archiveHandler: archiveHandler,
		cleaner:        cleaner,
	}
}

func (s *SupportArchiveReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info(fmt.Sprintf("Reconciler is triggered by resource %q", req.NamespacedName))

	archiveInterface := s.client.SupportArchives(req.Namespace)
	cr, err := archiveInterface.Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !slices.Contains(cr.GetFinalizers(), finalizerName) {
		logger.Info(fmt.Sprintf("Adding finalizer to support archive %q", cr.Name))
		cr.ObjectMeta.Finalizers = append(cr.ObjectMeta.Finalizers, finalizerName)
		_, updateErr := archiveInterface.Update(ctx, cr, metav1.UpdateOptions{})
		return ctrl.Result{Requeue: true}, updateErr
	}

	if !cr.GetDeletionTimestamp().IsZero() {
		cleanupErr := s.cleaner.Clean(ctx, cr.Name, cr.Namespace)
		if cleanupErr != nil {
			// Do not return here to avoid blocking in error case.
			// Garbage collection can try to clean up inconsistent files later.
			logger.Info(fmt.Sprintf("Failed to clean up for support archive request %q: %v", cr.Name, cleanupErr))
		}

		_, updateErr := archiveInterface.RemoveFinalizer(ctx, cr, finalizerName)
		if updateErr != nil {
			return ctrl.Result{Requeue: true}, updateErr
		}

		return ctrl.Result{Requeue: false}, nil
	}

	// Do not recreate archives
	if cr.Status.Phase == libv1.StatusPhaseCreated {
		return ctrl.Result{}, nil
	}

	requeue, err := s.archiveHandler.HandleArchiveRequest(ctx, cr)
	return ctrl.Result{Requeue: requeue}, err
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
		For(&libv1.SupportArchive{}).
		Complete(s)
}
