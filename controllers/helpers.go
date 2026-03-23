package controllers
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




































}	return types.NamespacedName{Namespace: namespace, Name: name}func namespacedName(namespace, name string) types.NamespacedName {}	return out	}		}			out = append(out, item)		if item != target {	for _, item := range items {	out := make([]string, 0, len(items))func removeString(items []string, target string) []string {}	return sets.NewString(items...).Has(target)func containsString(items []string, target string) bool {}	return out	}		out = append(out, newCondition)	if !found {	}		out = append(out, cond)		}			continue			out = append(out, newCondition)			}				newCondition.LastTransitionTime = cond.LastTransitionTime			if cond.Status == newCondition.Status && cond.Reason == newCondition.Reason && cond.Message == newCondition.Message {			found = true		if cond.Type == newCondition.Type {	for _, cond := range existing {	found := false	out := make([]metav1.Condition, 0, len(existing)+1)