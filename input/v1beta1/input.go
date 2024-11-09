// Package v1beta1 contains the input type for this Function
// +kubebuilder:object:generate=true
// +groupName=template.fn.crossplane.io
// +versionName=v1beta1
package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UXType is a custom type for the UX field.
type UXType string

const (
	// ClassicUX simulates the old/classic function experience before claim communications was added.
	ClassicUX UXType = "classic"
	// ModernUX represents the new/modern user experience that supports claim communications.
	ModernUX UXType = "modern"
)

// This isn't a custom resource, in the sense that we never install its CRD.
// It is a KRM-like object, so we generate a CRD to describe its schema.

// TODO: Add your input type here! It doesn't need to be called 'Input', you can
// rename it to anything you like.

// Input can be used to provide input to this Function.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:categories=crossplane
// +kubebuilder:validation:Enum="classic";"modern"
type Input struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// UX enables the selection of which user experience this function will provide.
	UX UXType `json:"ux,omitempty"`
}
