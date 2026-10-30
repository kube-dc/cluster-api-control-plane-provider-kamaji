package controllers

import (
	"context"
	"encoding/json"
	"testing"

	kcp "github.com/clastix/cluster-api-control-plane-provider-kamaji/api/v1alpha2"
	kamaji "github.com/clastix/kamaji/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	capi "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestKubeDCPartialProbeOverridesAndProxyPatchesReachTenant(t *testing.T) {
	for _, component := range []string{"apiServer", "controllerManager", "scheduler", "deployment"} {
		t.Run(component, func(t *testing.T) {
			scheme := runtime.NewScheme()
			for _, add := range []func(*runtime.Scheme) error{corev1.AddToScheme, kcp.AddToScheme, kamaji.AddToScheme, capi.AddToScheme} {
				if err := add(scheme); err != nil {
					t.Fatal(err)
				}
			}
			client := fake.NewClientBuilder().WithScheme(scheme).Build()
			reconciler := &KamajiControlPlaneReconciler{client: client}
			input := map[string]any{
				"apiVersion": "controlplane.cluster.x-k8s.io/v1alpha2", "kind": "KamajiControlPlane",
				"metadata": map[string]any{"name": "tenant-cp", "namespace": "tenant", "uid": "control-plane-uid"},
				"spec": map[string]any{"version": "v1.35.9", "replicas": 1,
					"network": map[string]any{"serviceType": "ClusterIP"},
					"addons": map[string]any{"kubeProxy": map[string]any{"enabled": true,
						"configurationJSONPatches": []any{map[string]any{"op": "add", "path": "/metricsBindAddress", "value": "0.0.0.0:10249"}}}},
					component: map[string]any{"probes": map[string]any{"readiness": map[string]any{"failureThreshold": 4, "timeoutSeconds": 10}}}},
			}
			raw, err := json.Marshal(input)
			if err != nil {
				t.Fatal(err)
			}
			var provider kcp.KamajiControlPlane
			if err := json.Unmarshal(raw, &provider); err != nil {
				t.Fatal(err)
			}
			cluster := capi.Cluster{}
			cluster.Spec.ClusterNetwork.APIServerPort = 6443
			tenant, err := reconciler.createOrUpdateTenantControlPlane(context.Background(), nil, cluster, provider)
			if err != nil {
				t.Fatal(err)
			}
			probes := tenant.Spec.ControlPlane.Deployment.Probes
			if probes == nil {
				t.Fatal("a single component's overrides were dropped")
			}
			selected := probes.Readiness
			switch component {
			case "apiServer":
				selected = probes.APIServer.Readiness
			case "controllerManager":
				selected = probes.ControllerManager.Readiness
			case "scheduler":
				selected = probes.Scheduler.Readiness
			}
			if selected == nil || *selected.FailureThreshold != 4 || *selected.TimeoutSeconds != 10 {
				t.Fatal("probe overrides changed during provider translation")
			}
			if len(tenant.Spec.Addons.KubeProxy.ConfigurationJSONPatches) != 1 || len(tenant.OwnerReferences) != 1 || tenant.OwnerReferences[0].UID != provider.UID {
				t.Fatal("proxy patches or provider ownership were dropped")
			}
			provider.Spec.ApiServer.Probes = nil
			provider.Spec.ControllerManager.Probes = nil
			provider.Spec.Scheduler.Probes = nil
			provider.Spec.Deployment.Probes = nil
			tenant, err = reconciler.createOrUpdateTenantControlPlane(context.Background(), nil, cluster, provider)
			if err != nil || tenant.Spec.ControlPlane.Deployment.Probes != nil {
				t.Fatalf("removed overrides remain on the existing tenant: %v", err)
			}
		})
	}
}
