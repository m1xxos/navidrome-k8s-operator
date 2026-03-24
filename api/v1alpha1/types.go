package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	PlaylistFinalizer = "playlist.navidrome.m1xxos.dev/finalizer"
	TrackFinalizer    = "track.navidrome.m1xxos.dev/finalizer"
)

type PlaylistSpec struct {
	NavidromeURL string `json:"navidromeURL"`
	Name         string `json:"name"`
	AuthSecret   string `json:"authSecret"`
}

type PlaylistStatus struct {
	RemotePlaylistID   string             `json:"remotePlaylistID,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

type Playlist struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PlaylistSpec   `json:"spec,omitempty"`
	Status PlaylistStatus `json:"status,omitempty"`
}

type PlaylistList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Playlist `json:"items"`
}

type TrackRef struct {
	TrackID  string `json:"trackID,omitempty"`
	FilePath string `json:"filePath,omitempty"`
	Artist   string `json:"artist,omitempty"`
	Title    string `json:"title,omitempty"`
}

type PlaylistRef struct {
	Name string `json:"name"`
}

type TrackSpec struct {
	PlaylistRef PlaylistRef `json:"playlistRef"`
	TrackRef    TrackRef    `json:"trackRef"`
	Priority    *int        `json:"priority,omitempty"`
	Position    int         `json:"position,omitempty"`
}

type TrackStatus struct {
	ResolvedTrackID    string             `json:"resolvedTrackID,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Synced             bool               `json:"synced,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

type Track struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TrackSpec   `json:"spec,omitempty"`
	Status TrackStatus `json:"status,omitempty"`
}

type TrackList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Track `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Playlist{}, &PlaylistList{}, &Track{}, &TrackList{})
}

func (in *Playlist) DeepCopyInto(out *Playlist) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
	if in.Status.Conditions != nil {
		out.Status.Conditions = make([]metav1.Condition, len(in.Status.Conditions))
		copy(out.Status.Conditions, in.Status.Conditions)
	}
}

func (in *Playlist) DeepCopy() *Playlist {
	if in == nil {
		return nil
	}
	out := new(Playlist)
	in.DeepCopyInto(out)
	return out
}

func (in *Playlist) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *PlaylistList) DeepCopyInto(out *PlaylistList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		out.Items = make([]Playlist, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}

func (in *PlaylistList) DeepCopy() *PlaylistList {
	if in == nil {
		return nil
	}
	out := new(PlaylistList)
	in.DeepCopyInto(out)
	return out
}

func (in *PlaylistList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *Track) DeepCopyInto(out *Track) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
	if in.Status.Conditions != nil {
		out.Status.Conditions = make([]metav1.Condition, len(in.Status.Conditions))
		copy(out.Status.Conditions, in.Status.Conditions)
	}
}

func (in *Track) DeepCopy() *Track {
	if in == nil {
		return nil
	}
	out := new(Track)
	in.DeepCopyInto(out)
	return out
}

func (in *Track) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *TrackList) DeepCopyInto(out *TrackList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		out.Items = make([]Track, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}

func (in *TrackList) DeepCopy() *TrackList {
	if in == nil {
		return nil
	}
	out := new(TrackList)
	in.DeepCopyInto(out)
	return out
}

func (in *TrackList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
