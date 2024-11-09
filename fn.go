package main

import (
	"context"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/logging"

	fnv1 "github.com/crossplane/function-sdk-go/proto/v1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/resource/composed"
	"github.com/crossplane/function-sdk-go/response"

	nopapis "github.com/crossplane-contrib/provider-nop/apis"
	nopv1alpha1 "github.com/crossplane-contrib/provider-nop/apis/v1alpha1"
	"github.com/jbw976/demo-xfn-claim-com/input/v1beta1"
)

// Function returns whatever response you ask it to.
type Function struct {
	fnv1.UnimplementedFunctionRunnerServiceServer
	log logging.Logger
}

// map of supported tiers for each environment
var supportedTiersByEnv = map[string][]string{
	"dev":        {"standard", "low"},
	"staging":    {"critical", "standard", "low"},
	"production": {"critical", "standard"},
}

// RunFunction processes the function request and generates landing zone
// resources for a given environment. If the provided tier is not supported for
// the given environment, the function will surface this as an error. This
// function is capable of the nice modern UX that surfaces these errors up to
// the claim level, but also simulating the classic old UX that does not surface
// any useful information to the user.
func (f *Function) RunFunction(_ context.Context, req *fnv1.RunFunctionRequest) (*fnv1.RunFunctionResponse, error) {
	f.log.Info("Running function", "tag", req.GetMeta().GetTag())

	rsp := response.To(req, response.DefaultTTL)

	// get the function input from the request
	in := &v1beta1.Input{}
	if err := request.GetInput(req, in); err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot get Function input from %T", req))
		return rsp, nil
	}

	// get the observed XR so we can read user specified config from it
	oxr, err := request.GetObservedCompositeResource(req)
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot get desired XR"))
		return rsp, nil
	}

	// retrieve relevant specified config from the XR
	env, _ := oxr.Resource.GetString("spec.environment")
	tier, _ := oxr.Resource.GetString("spec.tier")

	// check the forbidden combinations of env and tier
	supportedTiers, ok := supportedTiersByEnv[env]
	if !ok {
		response.Fatal(rsp, errors.Wrapf(err, "unsupported environment: %q", env))
		return rsp, nil
	}
	found := false
	for _, t := range supportedTiers {
		if t == tier {
			found = true
			break
		}
	}

	ready := corev1.ConditionTrue
	var envTierErr error
	if !found {
		// this tier is not supported for this environment, create an error to capture this invalid state
		ready = corev1.ConditionFalse
		envTierErr = errors.Errorf(
			"environment '%s' does not support tier '%s'. supported tiers for '%s' include '%s'",
			env, tier, env, strings.Join(supportedTiers, ", "))
	}

	// generate the desired composed resources for the landing zone. If we had
	// an error above due to invalid env/tier combination, we will also reflect
	// that in the ready status of the generated resources
	desiredResources, err := generateDesiredResources(
		[]string{"account", "user", "role", "security-group", "gateway", "cluster"},
		oxr.Resource.GetGenerateName(),
		ready)
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot generate desired composed resources on %T", rsp))
		return rsp, nil
	}

	err = response.SetDesiredComposedResources(rsp, desiredResources)
	if err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot set desired composed resources on %T", rsp))
		return rsp, nil
	}

	switch in.UX {
	case v1beta1.ClassicUX:
		// we should simulate the classic UX that does not surface any useful information to the user
		return handleClassicUX(rsp, envTierErr)
	case v1beta1.ModernUX:
		// use the nice new modern UX that communicates errors/conditions up to the claim level
		return handleModernUX(rsp, envTierErr)
	default:
		response.Fatal(rsp, errors.Errorf("unknown UX %q", in.UX))
		return rsp, nil
	}
}

func handleClassicUX(rsp *fnv1.RunFunctionResponse, err error) (*fnv1.RunFunctionResponse, error) {
	// in the classic/old UX, we would not surface any useful information to the
	// user, so don't use any of the nice .TargetComposite* methods
	if err != nil {
		response.Warning(rsp, errors.Wrapf(err, "landing zone provisioning failed"))
		response.ConditionFalse(rsp, "ProvisioningSuccess", "Error").WithMessage(err.Error())
		return rsp, nil
	}

	response.ConditionTrue(rsp, "ProvisioningSuccess", "Success")
	return rsp, nil
}

func handleModernUX(rsp *fnv1.RunFunctionResponse, err error) (*fnv1.RunFunctionResponse, error) {
	// in the modern UX, we communicate relevant and useful error information up
	// to the claim so the user can see it
	if err != nil {
		response.Warning(rsp, errors.Wrapf(err, "landing zone provisioning failed")).TargetCompositeAndClaim()
		response.ConditionFalse(rsp, "ProvisioningSuccess", "Error").WithMessage(err.Error()).TargetCompositeAndClaim()
		return rsp, nil
	}

	response.ConditionTrue(rsp, "ProvisioningSuccess", "Success").TargetCompositeAndClaim()
	return rsp, nil
}

func generateDesiredResources(names []string, baseName string, ready corev1.ConditionStatus) (map[resource.Name]*resource.DesiredComposed, error) {
	// Add the provider-nop v1alpha1 types to the composed resource scheme.
	// composed.From uses this to automatically set apiVersion and kind.
	_ = nopapis.AddToScheme(composed.Scheme)

	dcr := make(map[resource.Name]*resource.DesiredComposed, len(names))

	for _, name := range names {
		r := &nopv1alpha1.NopResource{
			ObjectMeta: metav1.ObjectMeta{
				Name: baseName + name,
			},
			Spec: nopv1alpha1.NopResourceSpec{
				ForProvider: nopv1alpha1.NopResourceParameters{
					// include a condition that will be ready true/false in 1
					// second depending on the input to this function
					ConditionAfter: []nopv1alpha1.ResourceConditionAfter{
						{Time: metav1.Duration{Duration: 1 * time.Second}, ConditionType: xpv1.TypeReady, ConditionStatus: ready},
					},
				},
			},
		}

		dcResource, err := composed.From(r)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot convert %T to %T", r, &composed.Unstructured{})
		}

		dcr[resource.Name(name)] = &resource.DesiredComposed{Resource: dcResource}
	}

	return dcr, nil
}
