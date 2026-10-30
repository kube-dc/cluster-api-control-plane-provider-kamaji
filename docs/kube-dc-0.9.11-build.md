# Build the Kube-DC 0.9.11 provider candidate

This fork is based on upstream control-plane provider 0.21.0 and serves
KamajiControlPlane v1alpha2. It retains Kube-DC's partial probe overrides and
kube-proxy JSON patches. Do not apply these CRDs over a v1alpha1 installation
without a separate migration qualification.

The complete package suite passes with Kubernetes 1.35 envtest. Contract tests
exercise each partial override, removal of overrides, tenant owner UIDs and
proxy patches on creation and update. Regenerate CRDs with controller-gen
0.20.1; scan only API packages for CRDs and deepcopy generation.

The module uses the reviewed Kube-DC Kamaji fork through `../kamaji`. Build with
that exact checkout as a named Docker context:

```sh
docker build --build-context kamaji-src=../kamaji -t IMAGE:UNUSED_TAG .
```

Record the provider and Kamaji source revisions with the image. Pin the
published digest in the fresh-install Fleet profile and select v1alpha2 in
k8-manager. Artifact scanning and live lifecycle acceptance remain required
before stable promotion.
