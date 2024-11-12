package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/utils/ptr"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/response"
)

func TestRunFunction(t *testing.T) {
	type args struct {
		ctx context.Context
		req *fnv1.RunFunctionRequest
	}
	type want struct {
		rsp *fnv1.RunFunctionResponse
		err error
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"UnsupportedTierClassicUX": {
			reason: "The Function should return no helpful conditions/events to the claim level in the classic UX",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "UnsupportedTierClassicUX"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "template.fn.crossplane.io/v1beta1",
						"kind": "Input",
						"ux": "classic"
					}`),
					Observed: &fnv1.State{
						Composite: &fnv1.Resource{
							Resource: resource.MustStructJSON(`{
								"apiVersion": "xp-demo.crossplane.io/v1alpha1",
								"kind": "XLandingZone",
								"spec": {
									"team": "core",
									"environment": "production",
									"tier": "low"
								}
							}`),
						},
					},
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Meta: &fnv1.ResponseMeta{Tag: "UnsupportedTierClassicUX", Ttl: durationpb.New(response.DefaultTTL)},
					Results: []*fnv1.Result{
						{
							Severity: fnv1.Severity_SEVERITY_WARNING,
							Message:  "landing zone provisioning failed: environment 'production' does not support tier 'low'. supported tiers for 'production' include 'critical, standard'",
							Target:   fnv1.Target_TARGET_COMPOSITE.Enum(),
						},
					},
					Conditions: []*fnv1.Condition{
						{
							Type:    "ProvisioningSuccess",
							Status:  fnv1.Status_STATUS_CONDITION_FALSE,
							Reason:  "Error",
							Message: ptr.To("environment 'production' does not support tier 'low'. supported tiers for 'production' include 'critical, standard'"),
							Target:  fnv1.Target_TARGET_COMPOSITE.Enum(),
						},
					},
					Desired: &fnv1.State{
						Resources: map[string]*fnv1.Resource{
							"account":        {Resource: generateTestResource("account", "False")},
							"user":           {Resource: generateTestResource("user", "False")},
							"role":           {Resource: generateTestResource("role", "False")},
							"security-group": {Resource: generateTestResource("security-group", "False")},
							"gateway":        {Resource: generateTestResource("gateway", "False")},
							"cluster":        {Resource: generateTestResource("cluster", "False")},
						},
					},
				},
			},
		},
		"UnsupportedTierModernUX": {
			reason: "The Function should return helpful conditions/events to the claim level in the modern UX",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "UnsupportedTierModernUX"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "template.fn.crossplane.io/v1beta1",
						"kind": "Input",
						"ux": "modern"
					}`),
					Observed: &fnv1.State{
						Composite: &fnv1.Resource{
							Resource: resource.MustStructJSON(`{
								"apiVersion": "xp-demo.crossplane.io/v1alpha1",
								"kind": "XLandingZone",
								"spec": {
									"team": "core",
									"environment": "production",
									"tier": "low"
								}
							}`),
						},
					},
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Meta: &fnv1.ResponseMeta{Tag: "UnsupportedTierModernUX", Ttl: durationpb.New(response.DefaultTTL)},
					Results: []*fnv1.Result{
						{
							Severity: fnv1.Severity_SEVERITY_WARNING,
							Message:  "landing zone provisioning failed: environment 'production' does not support tier 'low'. supported tiers for 'production' include 'critical, standard'",
							Target:   fnv1.Target_TARGET_COMPOSITE_AND_CLAIM.Enum(),
						},
					},
					Conditions: []*fnv1.Condition{
						{
							Type:    "ProvisioningSuccess",
							Status:  fnv1.Status_STATUS_CONDITION_FALSE,
							Reason:  "Error",
							Message: ptr.To("environment 'production' does not support tier 'low'. supported tiers for 'production' include 'critical, standard'"),
							Target:  fnv1.Target_TARGET_COMPOSITE_AND_CLAIM.Enum(),
						},
					},
					Desired: &fnv1.State{
						Resources: map[string]*fnv1.Resource{
							"account":        {Resource: generateTestResource("account", "False")},
							"user":           {Resource: generateTestResource("user", "False")},
							"role":           {Resource: generateTestResource("role", "False")},
							"security-group": {Resource: generateTestResource("security-group", "False")},
							"gateway":        {Resource: generateTestResource("gateway", "False")},
							"cluster":        {Resource: generateTestResource("cluster", "False")},
						},
					},
				},
			},
		},
		"SupportedTierClassicUX": {
			reason: "The Function should not return success to the claim level when a supported tier is provided in the classic UX",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "SupportedTierClassicUX"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "template.fn.crossplane.io/v1beta1",
						"kind": "Input",
						"ux": "classic"
					}`),
					Observed: &fnv1.State{
						Composite: &fnv1.Resource{
							Resource: resource.MustStructJSON(`{
								"apiVersion": "xp-demo.crossplane.io/v1alpha1",
								"kind": "XLandingZone",
								"spec": {
									"team": "core",
									"environment": "production",
									"tier": "critical"
								}
							}`),
						},
					},
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Meta: &fnv1.ResponseMeta{Tag: "SupportedTierClassicUX", Ttl: durationpb.New(response.DefaultTTL)},
					Conditions: []*fnv1.Condition{
						{
							Type:   "ProvisioningSuccess",
							Status: fnv1.Status_STATUS_CONDITION_TRUE,
							Reason: "Success",
							Target: fnv1.Target_TARGET_COMPOSITE.Enum(),
						},
					},
					Desired: &fnv1.State{
						Resources: map[string]*fnv1.Resource{
							"account":        {Resource: generateTestResource("account", "True")},
							"user":           {Resource: generateTestResource("user", "True")},
							"role":           {Resource: generateTestResource("role", "True")},
							"security-group": {Resource: generateTestResource("security-group", "True")},
							"gateway":        {Resource: generateTestResource("gateway", "True")},
							"cluster":        {Resource: generateTestResource("cluster", "True")},
						},
					},
				},
			},
		},
		"SupportedTierModernUX": {
			reason: "The Function should return success to the claim level when a supported tier is provided in the modern UX",
			args: args{
				req: &fnv1.RunFunctionRequest{
					Meta: &fnv1.RequestMeta{Tag: "SupportedTierModernUX"},
					Input: resource.MustStructJSON(`{
						"apiVersion": "template.fn.crossplane.io/v1beta1",
						"kind": "Input",
						"ux": "modern"
					}`),
					Observed: &fnv1.State{
						Composite: &fnv1.Resource{
							Resource: resource.MustStructJSON(`{
								"apiVersion": "xp-demo.crossplane.io/v1alpha1",
								"kind": "XLandingZone",
								"spec": {
									"team": "core",
									"environment": "production",
									"tier": "critical"
								}
							}`),
						},
					},
				},
			},
			want: want{
				rsp: &fnv1.RunFunctionResponse{
					Meta: &fnv1.ResponseMeta{Tag: "SupportedTierModernUX", Ttl: durationpb.New(response.DefaultTTL)},
					Conditions: []*fnv1.Condition{
						{
							Type:   "ProvisioningSuccess",
							Status: fnv1.Status_STATUS_CONDITION_TRUE,
							Reason: "Success",
							Target: fnv1.Target_TARGET_COMPOSITE_AND_CLAIM.Enum(),
						},
					},
					Desired: &fnv1.State{
						Resources: map[string]*fnv1.Resource{
							"account":        {Resource: generateTestResource("account", "True")},
							"user":           {Resource: generateTestResource("user", "True")},
							"role":           {Resource: generateTestResource("role", "True")},
							"security-group": {Resource: generateTestResource("security-group", "True")},
							"gateway":        {Resource: generateTestResource("gateway", "True")},
							"cluster":        {Resource: generateTestResource("cluster", "True")},
						},
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := &Function{log: logging.NewNopLogger()}
			rsp, err := f.RunFunction(tc.args.ctx, tc.args.req)

			if diff := cmp.Diff(tc.want.rsp, rsp, protocmp.Transform()); diff != "" {
				t.Errorf("%s\nf.RunFunction(...): -want rsp, +got rsp:\n%s", tc.reason, diff)
			}

			if diff := cmp.Diff(tc.want.err, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("%s\nf.RunFunction(...): -want err, +got err:\n%s", tc.reason, diff)
			}
		})
	}
}

func generateTestResource(name, status string) *structpb.Struct {
	return resource.MustStructJSON(fmt.Sprintf(`{
		"apiVersion": "nop.crossplane.io/v1alpha1",
		"kind": "NopResource",
		"metadata": {
			"name": "%s"
		},
		"spec": {
			"forProvider": {
				"conditionAfter": [
					{
						"conditionStatus": "%s",
						"conditionType": "Ready",
						"time": "1s"
					}
				]
			}
		},
		"status": {
			"observedGeneration": 0
		}
	}`, name, status))
}
