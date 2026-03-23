package controllers
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


























































































































}		Complete(r)		For(&navv1alpha1.Playlist{}).	return ctrl.NewControllerManagedBy(mgr).func (r *PlaylistReconciler) SetupWithManager(mgr ctrl.Manager) error {}	return user, pass, nil	}		return "", "", fmt.Errorf("secret %s/%s must contain username and password", namespace, secretName)	if user == "" || pass == "" {	pass := string(secret.Data["password"])	user := string(secret.Data["username"])	}		return "", "", err	if err := r.Get(ctx, namespacedName(namespace, secretName), secret); err != nil {	secret := &corev1.Secret{}func (r *PlaylistReconciler) readCredentials(ctx context.Context, namespace, secretName string) (string, string, error) {}	return ctrl.Result{RequeueAfter: 20 * time.Second}, nil	r.Recorder.Event(playlist, corev1.EventTypeWarning, reason, message)	_ = r.Status().Update(ctx, playlist)	})		Message: message,		Reason:  reason,		Status:  metav1.ConditionFalse,		Type:    "Ready",	playlist.Status.Conditions = setCondition(playlist.Status.Conditions, metav1.Condition{	playlist.Status.ObservedGeneration = playlist.Generationfunc (r *PlaylistReconciler) failPlaylistStatus(ctx context.Context, playlist *navv1alpha1.Playlist, reason, message string) (ctrl.Result, error) {}	return ctrl.Result{}, nil	}		return ctrl.Result{}, err	if err := r.Update(ctx, playlist); err != nil {	playlist.Finalizers = removeString(playlist.Finalizers, navv1alpha1.PlaylistFinalizer)	}		}			}				}					return ctrl.Result{}, delErr				if delErr := navClient.DeletePlaylist(ctx, playlist.Status.RemotePlaylistID); delErr != nil {			if loginErr := navClient.Login(ctx, user, pass); loginErr == nil {			navClient := r.NavClientFactory.New(playlist.Spec.NavidromeURL)		if err == nil {		user, pass, err := r.readCredentials(ctx, playlist.Namespace, playlist.Spec.AuthSecret)	if playlist.Status.RemotePlaylistID != "" && playlist.Spec.NavidromeURL != "" && playlist.Spec.AuthSecret != "" {	}		return ctrl.Result{}, nil	if !containsString(playlist.Finalizers, navv1alpha1.PlaylistFinalizer) {func (r *PlaylistReconciler) handleDelete(ctx context.Context, playlist *navv1alpha1.Playlist) (ctrl.Result, error) {}	return ctrl.Result{RequeueAfter: 10 * time.Minute}, nil	logger.Info("playlist synced", "remotePlaylistID", remoteID)	r.Recorder.Event(playlist, corev1.EventTypeNormal, "Synced", "Playlist synced with Navidrome")	}		return ctrl.Result{}, err	if err := r.Status().Update(ctx, playlist); err != nil {	})		Message: fmt.Sprintf("Playlist %q synced with Navidrome", playlist.Spec.Name),		Reason:  "Synced",		Status:  metav1.ConditionTrue,		Type:    "Ready",	playlist.Status.Conditions = setCondition(playlist.Status.Conditions, metav1.Condition{	playlist.Status.ObservedGeneration = playlist.Generation	playlist.Status.RemotePlaylistID = remoteID	}		return r.failPlaylistStatus(ctx, playlist, "SyncFailed", err.Error())	if err != nil {	remoteID, err := navClient.EnsurePlaylist(ctx, playlist.Spec.Name)	}		return r.failPlaylistStatus(ctx, playlist, "AuthFailed", err.Error())	if err := navClient.Login(ctx, user, pass); err != nil {	navClient := r.NavClientFactory.New(playlist.Spec.NavidromeURL)	}		return r.failPlaylistStatus(ctx, playlist, "AuthSecretError", err.Error())	if err != nil {	user, pass, err := r.readCredentials(ctx, playlist.Namespace, playlist.Spec.AuthSecret)	}		}			return ctrl.Result{}, err		if err := r.Update(ctx, playlist); err != nil {		playlist.Finalizers = append(playlist.Finalizers, navv1alpha1.PlaylistFinalizer)	if !containsString(playlist.Finalizers, navv1alpha1.PlaylistFinalizer) {	}		return r.handleDelete(ctx, playlist)	if !playlist.DeletionTimestamp.IsZero() {	}		return r.failPlaylistStatus(ctx, playlist, "SpecInvalid", "spec.navidromeURL/spec.name/spec.authSecret are required")	if playlist.Spec.NavidromeURL == "" || playlist.Spec.Name == "" || playlist.Spec.AuthSecret == "" {	}		return ctrl.Result{}, err		}			return ctrl.Result{}, nil		if apierrors.IsNotFound(err) {	if err := r.Get(ctx, req.NamespacedName, playlist); err != nil {	playlist := &navv1alpha1.Playlist{}	logger := log.FromContext(ctx).WithValues("playlist", req.NamespacedName)func (r *PlaylistReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {}	NavClientFactory navidrome.ClientFactory	Recorder         record.EventRecorder	Scheme *runtime.Scheme	client.Clienttype PlaylistReconciler struct {