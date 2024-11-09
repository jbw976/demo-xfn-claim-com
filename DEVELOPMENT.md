# Local Development

## Build functions and providers

### Build function `demo-xfn-claim-com`

Build the function images and packages:
```
docker build . --platform=linux/amd64 --tag demo-xfn-claim-com-runtime-amd64
docker build . --platform=linux/arm64 --tag demo-xfn-claim-com-runtime-arm64

crossplane xpkg build \
    --package-root=package \
    --embed-runtime-image=demo-xfn-claim-com-runtime-amd64 \
    --package-file=function-amd64.xpkg
crossplane xpkg build \
    --package-root=package \
    --embed-runtime-image=demo-xfn-claim-com-runtime-arm64 \
    --package-file=function-arm64.xpkg
```

### Build `function-auto-ready`

Build the function images and packages:
```
docker build . --platform=linux/amd64 --tag auto-ready-runtime-amd64
docker build . --platform=linux/arm64 --tag auto-ready-runtime-arm64

crossplane xpkg build \
    --package-root=package \
    --embed-runtime-image=auto-ready-runtime-amd64 \
    --package-file=function-amd64.xpkg
crossplane xpkg build \
    --package-root=package \
    --embed-runtime-image=auto-ready-runtime-arm64 \
    --package-file=function-arm64.xpkg
```

### Build `provider-nop`

Build the provider image:
```
PROVIDER_NOP_VERSION=v0.0.3
VERSION=${PROVIDER_NOP_VERSION} make build.all
```

## Create local cluster with a package cache

Create a kind cluster that will support a package cache:
```
kind create cluster --config=package-cache/kind.yaml
```

Then create the PV/PVC to back the package cache and install Crossplane to use this package cache:
```
kubectl apply -f package-cache/pv.yaml
kubectl create ns crossplane-system
kubectl apply -f package-cache/pvc.yaml
helm install crossplane --namespace crossplane-system --create-namespace crossplane-stable/crossplane --set packageCache.pvc=package-cache
```

## Load local packages

Load the function package and runtime image into the cluster:
```
docker tag demo-xfn-claim-com-runtime-amd64 xpkg.upbound.io/demo-xfn-claim-com
kind load docker-image xpkg.upbound.io/demo-xfn-claim-com
up xpkg xp-extract --from-xpkg function-amd64.xpkg -o ~/dev/package-cache/demo-xfn-claim-com.gz && chmod 644 ~/dev/package-cache/demo-xfn-claim-com.gz
```

Load function-auto-ready package and runtime image into the cluster:
```
docker tag auto-ready-runtime-amd64 xpkg.upbound.io/function-auto-ready
kind load docker-image xpkg.upbound.io/function-auto-ready
up xpkg xp-extract --from-xpkg function-amd64.xpkg -o ~/dev/package-cache/function-auto-ready.gz && chmod 644 ~/dev/package-cache/function-auto-ready.gz
```

Load provider-nop local package:
```
docker tag $(docker images | grep provider-nop-amd64 | awk '{print $1}') xpkg.upbound.io/provider-nop
kind load docker-image xpkg.upbound.io/provider-nop
up xpkg xp-extract --from-xpkg _output/xpkg/linux_amd64/provider-nop-${PROVIDER_NOP_VERSION}.xpkg -o ~/dev/package-cache/provider-nop.gz && chmod 644 ~/dev/package-cache/provider-nop.gz
```

## Install packages

Install the function package from the local package cache:
```
kubectl apply -f package-cache/function.yaml
kubectl apply -f package-cache/packages.yaml
```

## Create `LandingZone` Platform API

Define your `LandingZone` abstraction by creating a `CompositeResourceDefinition` (XRD) that defines the API and a `Composition` that defines the implementation:
```
kubectl apply -f example/xrd.yaml
kubectl apply -f example/composition-classic.yaml
kubectl apply -f example/composition-modern.yaml
```

## Create `LandingZone` objects

Create a claim for this `LandingZone` abstraction:
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

### Local dev inner loop

Clean up the objects:
```
kubectl delete -f example/claim-classic.yaml
kubectl delete -f example/claim-modern.yaml
```

Clean up the function:
```
kubectl delete -f package-cache/function.yaml
```

The build and set up the testing scenario again:
* build/load function
* install function
* create objects
* check results again

## Pushing to a Registry

Push a new version of the Function package to the Marketplace:
```
export FUNCTION_VERSION="v0.0.3"
crossplane xpkg push \
  --package-files=function-amd64.xpkg,function-arm64.xpkg \
  xpkg.upbound.io/jaredorg/demo-xfn-claim-com:${FUNCTION_VERSION}
```