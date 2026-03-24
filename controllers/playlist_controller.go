package controllers

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	navv1alpha1 "github.com/m1xxos/navidrome-k8s-operator/api/v1alpha1"
	"github.com/m1xxos/navidrome-k8s-operator/internal/navidrome"
)

type PlaylistReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	Recorder         record.EventRecorder
	NavClientFactory navidrome.ClientFactory
}

func (r *PlaylistReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("playlist", req.NamespacedName)

	playlist := &navv1alpha1.Playlist{}
	if err := r.Get(ctx, req.NamespacedName, playlist); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if playlist.Spec.NavidromeURL == "" || playlist.Spec.Name == "" || playlist.Spec.AuthSecret == "" {
		return r.failPlaylistStatus(ctx, playlist, "SpecInvalid", "spec.navidromeURL/spec.name/spec.authSecret are required")
	}

	if !playlist.DeletionTimestamp.IsZero() {
		return r.handleDelete(ctx, playlist)
	}

	if !containsString(playlist.Finalizers, navv1alpha1.PlaylistFinalizer) {
		playlist.Finalizers = append(playlist.Finalizers, navv1alpha1.PlaylistFinalizer)
		if err := r.Update(ctx, playlist); err != nil {
			return ctrl.Result{}, err
		}
	}

	user, pass, err := r.readCredentials(ctx, playlist.Namespace, playlist.Spec.AuthSecret)
	if err != nil {
		return r.failPlaylistStatus(ctx, playlist, "AuthSecretError", err.Error())
	}

	navClient := r.NavClientFactory.New(playlist.Spec.NavidromeURL)
	if err := navClient.Login(ctx, user, pass); err != nil {
		return r.failPlaylistStatus(ctx, playlist, "AuthFailed", err.Error())
	}

	remoteID, err := navClient.EnsurePlaylist(ctx, playlist.Spec.Name)
	if err != nil {
		return r.failPlaylistStatus(ctx, playlist, "SyncFailed", err.Error())
	}

	readyMessage := fmt.Sprintf("Playlist %q synced with Navidrome", playlist.Spec.Name)
	if playlist.Status.RemotePlaylistID == remoteID && playlist.Status.ObservedGeneration == playlist.Generation {
		logger.V(1).Info("playlist already synced for current generation", "remotePlaylistID", remoteID)
		return ctrl.Result{RequeueAfter: 10 * time.Minute}, nil
	}

	playlist.Status.RemotePlaylistID = remoteID
	playlist.Status.ObservedGeneration = playlist.Generation
	playlist.Status.Conditions = setCondition(playlist.Status.Conditions, metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionTrue,
		Reason:  "Synced",
		Message: readyMessage,
	})
	if err := r.Status().Update(ctx, playlist); err != nil {
		return ctrl.Result{}, err
	}

	r.Recorder.Event(playlist, corev1.EventTypeNormal, "Synced", "Playlist synced with Navidrome")
	logger.Info("playlist synced", "remotePlaylistID", remoteID)
	return ctrl.Result{RequeueAfter: 10 * time.Minute}, nil
}

func (r *PlaylistReconciler) handleDelete(ctx context.Context, playlist *navv1alpha1.Playlist) (ctrl.Result, error) {
	if !containsString(playlist.Finalizers, navv1alpha1.PlaylistFinalizer) {
		return ctrl.Result{}, nil
	}

	if playlist.Status.RemotePlaylistID != "" && playlist.Spec.NavidromeURL != "" && playlist.Spec.AuthSecret != "" {
		user, pass, err := r.readCredentials(ctx, playlist.Namespace, playlist.Spec.AuthSecret)
		if err == nil {
			navClient := r.NavClientFactory.New(playlist.Spec.NavidromeURL)
			if loginErr := navClient.Login(ctx, user, pass); loginErr == nil {
				if delErr := navClient.DeletePlaylist(ctx, playlist.Status.RemotePlaylistID); delErr != nil {
					return ctrl.Result{}, delErr
				}
			}
		}
	}

	playlist.Finalizers = removeString(playlist.Finalizers, navv1alpha1.PlaylistFinalizer)
	if err := r.Update(ctx, playlist); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *PlaylistReconciler) failPlaylistStatus(ctx context.Context, playlist *navv1alpha1.Playlist, reason, message string) (ctrl.Result, error) {
	playlist.Status.ObservedGeneration = playlist.Generation
	playlist.Status.Conditions = setCondition(playlist.Status.Conditions, metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionFalse,
		Reason:  reason,
		Message: message,
	})
	_ = r.Status().Update(ctx, playlist)
	r.Recorder.Event(playlist, corev1.EventTypeWarning, reason, message)
	return ctrl.Result{RequeueAfter: 20 * time.Second}, nil
}

func (r *PlaylistReconciler) readCredentials(ctx context.Context, namespace, secretName string) (string, string, error) {
	secret := &corev1.Secret{}
	if err := r.Get(ctx, namespacedName(namespace, secretName), secret); err != nil {
		return "", "", err
	}
	user := string(secret.Data["username"])
	pass := string(secret.Data["password"])
	if user == "" || pass == "" {
		return "", "", fmt.Errorf("secret %s/%s must contain username and password", namespace, secretName)
	}
	return user, pass, nil
}

func (r *PlaylistReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&navv1alpha1.Playlist{}).
		Complete(r)
}
