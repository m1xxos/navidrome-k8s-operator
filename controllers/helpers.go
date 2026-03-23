package controllers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
)

func setCondition(conditions []metav1.Condition, condition metav1.Condition) []metav1.Condition {
	if condition.LastTransitionTime.IsZero() {
		condition.LastTransitionTime = metav1.Now()
	}
	return metaSetStatusCondition(conditions, condition)
}

func metaSetStatusCondition(existing []metav1.Condition, newCondition metav1.Condition) []metav1.Condition {
	out := make([]metav1.Condition, 0, len(existing)+1)
	found := false
	for _, cond := range existing {
		if cond.Type == newCondition.Type {
			found = true
			if cond.Status == newCondition.Status && cond.Reason == newCondition.Reason && cond.Message == newCondition.Message {
				newCondition.LastTransitionTime = cond.LastTransitionTime
			}
			out = append(out, newCondition)
			continue
		}
		out = append(out, cond)
	}
	if !found {
		out = append(out, newCondition)
	}
	return out
}

func containsString(items []string, target string) bool {
	return sets.NewString(items...).Has(target)
}

func removeString(items []string, target string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item != target {
			out = append(out, item)
		}
	}
	return out
}

func namespacedName(namespace, name string) types.NamespacedName {
	return types.NamespacedName{Namespace: namespace, Name: name}
}
