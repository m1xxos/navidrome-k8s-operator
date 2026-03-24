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

type TrackReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	Recorder         record.EventRecorder
	NavClientFactory navidrome.ClientFactory
}

func (r *TrackReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("track", req.NamespacedName)

	track := &navv1alpha1.Track{}
	if err := r.Get(ctx, req.NamespacedName, track); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if track.Spec.PlaylistRef.Name == "" {
		return r.failTrackStatus(ctx, track, "SpecInvalid", "spec.playlistRef.name is required")
	}

	playlist := &navv1alpha1.Playlist{}
	if err := r.Get(ctx, namespacedName(track.Namespace, track.Spec.PlaylistRef.Name), playlist); err != nil {
		if apierrors.IsNotFound(err) {
			return r.failTrackStatus(ctx, track, "PlaylistNotFound", "referenced playlist not found")
		}
		return ctrl.Result{}, err
	}

	if !track.DeletionTimestamp.IsZero() {
		logger.Info("Started reconciling Track deletion", "resolvedTrackID", track.Status.ResolvedTrackID, "order", trackOrder(track.Spec))
		return r.handleDelete(ctx, track, playlist)
	}

	if !containsString(track.Finalizers, navv1alpha1.TrackFinalizer) {
		track.Finalizers = append(track.Finalizers, navv1alpha1.TrackFinalizer)
		if err := r.Update(ctx, track); err != nil {
			return ctrl.Result{}, err
		}
	}

	if track.Status.Synced && track.Status.ObservedGeneration == track.Generation {
		logger.V(1).Info("Track was already synced", "generation", track.Generation)
		return ctrl.Result{}, nil
	}

	if playlist.Status.RemotePlaylistID == "" {
		return r.failTrackStatus(ctx, track, "PlaylistNotReady", "playlist has no remote ID yet")
	}

	user, pass, err := r.readCredentials(ctx, playlist.Namespace, playlist.Spec.AuthSecret)
	if err != nil {
		return r.failTrackStatus(ctx, track, "AuthSecretError", err.Error())
	}

	navClient := r.NavClientFactory.New(playlist.Spec.NavidromeURL)
	if err := navClient.Login(ctx, user, pass); err != nil {
		return r.failTrackStatus(ctx, track, "AuthFailed", err.Error())
	}

	resolvedTrackID, err := navClient.ResolveTrack(ctx, navidrome.TrackSelector{
		TrackID:  track.Spec.TrackRef.TrackID,
		FilePath: track.Spec.TrackRef.FilePath,
		Artist:   track.Spec.TrackRef.Artist,
		Title:    track.Spec.TrackRef.Title,
	})
	if err != nil {
		return r.failTrackStatus(ctx, track, "TrackResolveFailed", err.Error())
	}

	orderIndex := trackOrder(track.Spec)
	logger.Info("Syncing Track with Playlist", "remotePlaylistID", playlist.Status.RemotePlaylistID, "order", orderIndex)
	if err := navClient.AddOrMoveTrack(ctx, playlist.Status.RemotePlaylistID, resolvedTrackID, orderIndex); err != nil {
		return r.failTrackStatus(ctx, track, "SyncFailed", err.Error())
	}

	track.Status.ResolvedTrackID = resolvedTrackID
	track.Status.ObservedGeneration = track.Generation
	track.Status.Synced = true
	track.Status.Conditions = setCondition(track.Status.Conditions, metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionTrue,
		Reason:  "Synced",
		Message: fmt.Sprintf("Track synced in playlist %q", playlist.Spec.Name),
	})
	if err := r.Status().Update(ctx, track); err != nil {
		return ctrl.Result{}, err
	}

	r.Recorder.Event(track, corev1.EventTypeNormal, "Synced", "Track synced with playlist")
	logger.Info("Synced Track with Playlist", "resolvedTrackID", resolvedTrackID, "order", orderIndex, "generation", track.Generation)
	return ctrl.Result{}, nil
}

func (r *TrackReconciler) handleDelete(ctx context.Context, track *navv1alpha1.Track, playlist *navv1alpha1.Playlist) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("track", client.ObjectKeyFromObject(track))

	if !containsString(track.Finalizers, navv1alpha1.TrackFinalizer) {
		logger.V(1).Info("Track finalizer was already removed")
		return ctrl.Result{}, nil
	}

	if playlist != nil && playlist.Status.RemotePlaylistID != "" && track.Status.ResolvedTrackID != "" {
		user, pass, err := r.readCredentials(ctx, playlist.Namespace, playlist.Spec.AuthSecret)
		if err == nil {
			navClient := r.NavClientFactory.New(playlist.Spec.NavidromeURL)
			if loginErr := navClient.Login(ctx, user, pass); loginErr == nil {
				if rmErr := navClient.RemoveTrack(ctx, playlist.Status.RemotePlaylistID, track.Status.ResolvedTrackID, trackOrder(track.Spec)); rmErr != nil {
					return ctrl.Result{}, rmErr
				}
				logger.Info("Deleted Track from Playlist", "remotePlaylistID", playlist.Status.RemotePlaylistID, "resolvedTrackID", track.Status.ResolvedTrackID, "order", trackOrder(track.Spec))
			}
		}
	} else {
		logger.Info("Skipped Track remote cleanup", "hasPlaylist", playlist != nil, "hasRemotePlaylistID", playlist != nil && playlist.Status.RemotePlaylistID != "", "hasResolvedTrackID", track.Status.ResolvedTrackID != "")
	}

	track.Finalizers = removeString(track.Finalizers, navv1alpha1.TrackFinalizer)
	if err := r.Update(ctx, track); err != nil {
		return ctrl.Result{}, err
	}
	logger.Info("Removed Track finalizer")
	return ctrl.Result{}, nil
}

func (r *TrackReconciler) failTrackStatus(ctx context.Context, track *navv1alpha1.Track, reason, message string) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("track", client.ObjectKeyFromObject(track))
	logger.Error(fmt.Errorf("%s", message), "Failed to sync Track", "reason", reason)

	track.Status.ObservedGeneration = track.Generation
	track.Status.Synced = false
	track.Status.Conditions = setCondition(track.Status.Conditions, metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionFalse,
		Reason:  reason,
		Message: message,
	})
	_ = r.Status().Update(ctx, track)
	r.Recorder.Event(track, corev1.EventTypeWarning, reason, message)
	return ctrl.Result{RequeueAfter: 20 * time.Second}, nil
}

func (r *TrackReconciler) readCredentials(ctx context.Context, namespace, secretName string) (string, string, error) {
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

func (r *TrackReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&navv1alpha1.Track{}).
		Complete(r)
}

func trackOrder(spec navv1alpha1.TrackSpec) int {
	if spec.Priority != nil {
		if *spec.Priority < 0 {
			return 0
		}
		return *spec.Priority
	}
	if spec.Position < 0 {
		return 0
	}
	return spec.Position
}
