package reconciler

import (
	"context"
	"errors"
	"fmt"

	libv1 "github.com/cloudogu/k8s-support-archive-lib/api/v1"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"

	k8sErrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type SupportArchiveReconciler struct {
	client        supportArchiveV1Interface
	createHandler createArchiveHandler
	deleteHandler deleteArchiveHandler
}

func NewSupportArchiveReconciler(client supportArchiveV1Interface, createHandler createArchiveHandler, deleteHandler deleteArchiveHandler) *SupportArchiveReconciler {
	return &SupportArchiveReconciler{
		client:        client,
		createHandler: createHandler,
		deleteHandler: deleteHandler,
	}
}

func (s *SupportArchiveReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info(fmt.Sprintf("Reconciler is triggered by resource %q", req.NamespacedName))

	archiveInterface := s.client.SupportArchives(req.Namespace)
	cr, err := archiveInterface.Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil && !k8sErrs.IsNotFound(err) {
		return ctrl.Result{}, err
	} else if k8sErrs.IsNotFound(err) || !cr.GetDeletionTimestamp().IsZero() {
		cleanupErr := s.deleteHandler.Delete(ctx, domain.SupportArchiveID{
			Namespace: req.Namespace,
			Name:      req.Name,
		})
		return ctrl.Result{}, cleanupErr
	}

	requeue, err := s.createHandler.HandleArchiveRequest(ctx, cr)
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
