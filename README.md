# demo-xfn-claim-com

This repo contains an end to end demo for Crossplane Composition Functions that
utilizes the functionality to expose status and events up to the level of a
claim.

The demo will create two `LandingZone` resources that will fail, but demonstrate
how the UX has greatly improved.  The first `LandingZone` will use the
classic/old UX and the user will never see any useful error information to determine root cause.

The second `LandingZone` uses the modern Crossplane (v1.17+) UX and the user
will see an obvious root cause and how to fix it surfaced up to the claim.

## Development

Instructions for local development of this function can be found in [DEVELOPMENT.md](./DEVELOPMENT.md).

## Pre-Requisites

Create a Kubernetes cluster, e.g. with `kind`:
```
kind create cluster
```

Install Crossplane from the `stable` release channel, e.g.:
```
helm repo add crossplane-stable https://charts.crossplane.io/stable
helm repo update
helm install crossplane --namespace crossplane-system --create-namespace crossplane-stable/crossplane
```

## Setup

Install the `demo-xfn-claim-com` Function:
```
kubectl apply -f example/functions.yaml
kubectl apply -f example/providers.yaml
```

Wait for them to become installed/healthy:
```
kubectl get pkg
```

Define your `LandingZone` abstraction by creating a `CompositeResourceDefinition` (XRD) that defines the API and a `Composition` that defines the implementation:
```
kubectl apply -f example/xrd.yaml
kubectl apply -f example/composition-classic.yaml
kubectl apply -f example/composition-modern.yaml
```

## Demo

Create a claim for this `LandingZone` abstraction, using the classic/old UX:
```
kubectl apply -f example/claim-classic.yaml
```

We expect this claim to never become ready/healthy, but because it's using the classic UX, there won't be any good information on why or how to fix it.
```
kubectl get landingzone.xp-demo.crossplane.io/landing-zone-dev
kubectl get landingzone.xp-demo.crossplane.io/landing-zone-dev -ojson | jq '.status'
kubectl describe landingzone.xp-demo.crossplane.io/landing-zone-dev
```

Create another `LandingZone` claim that uses the modern UX this time:
```
kubectl apply -f example/claim-modern.yaml
```

We also expect this to fail, but this time it will be much more clear what the issue is because the function is communicating specific status and events back to the claim.
```
kubectl get landingzone.xp-demo.crossplane.io/landing-zone-prod
kubectl get landingzone.xp-demo.crossplane.io/landing-zone-prod -ojson | jq '.status'
kubectl describe landingzone.xp-demo.crossplane.io/landing-zone-prod
```

Voila! The root cause of this claim's failure is now clear and actionable because the function has communicated it up to the claim as a status condition and an event. Now we know how to fix this error by updating the claim's `.spec.tier` field to a valid value, such as `standard` or `critical`.

Make this edit, then apply the claim again:
```
kubectl apply -f example/claim-modern.yaml
```

We have fixed the error and we expect the claim to get to ready/healthy soon.
```
kubectl get landingzone.xp-demo.crossplane.io/landing-zone-prod
kubectl get landingzone.xp-demo.crossplane.io/landing-zone-prod -ojson | jq '.status'
```

## Clean up
Clean up all the objects:
```
kubectl delete -f example/claim-classic.yaml
kubectl delete -f example/claim-modern.yaml
```

Delete the compositions and XRD:
```
kubectl delete -f example/xrd.yaml
kubectl delete -f example/composition-classic.yaml
kubectl delete -f example/composition-modern.yaml
```

Uninstall the functions/providers:
```
kubectl delete -f example/functions.yaml
kubectl delete -f example/providers.yaml
```