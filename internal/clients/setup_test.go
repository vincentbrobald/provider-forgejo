// SPDX-FileCopyrightText: 2026 Vincent Brobald
// SPDX-License-Identifier: MPL-2.0

package clients

import (
	"context"
	"strings"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	clusterapis "brobald.be/provider-forgejo/apis/cluster"
	clusterrepo "brobald.be/provider-forgejo/apis/cluster/repository/v1alpha1"
	clusterv1beta1 "brobald.be/provider-forgejo/apis/cluster/v1beta1"
	namespacedapis "brobald.be/provider-forgejo/apis/namespaced"
	nsrepo "brobald.be/provider-forgejo/apis/namespaced/repository/v1alpha1"
	namespacedv1beta1 "brobald.be/provider-forgejo/apis/namespaced/v1beta1"
)

func sec(ns string, data map[string]string) *corev1.Secret {
	s := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "forgejo"}, Data: map[string][]byte{}}
	for k, v := range data {
		s.Data[k] = []byte(v)
	}
	return s
}

func secretRef(ns string) namespacedv1beta1.ProviderConfigSpec {
	spec := namespacedv1beta1.ProviderConfigSpec{}
	spec.Credentials.Source = xpv2.CredentialsSourceSecret
	spec.Credentials.SecretRef = &xpv2.SecretKeySelector{SecretReference: xpv2.SecretReference{Namespace: ns, Name: "forgejo"}, Key: "credentials"}
	return spec
}

func kubeWith(t *testing.T, objs ...client.Object) client.Client {
	t.Helper()
	s := runtime.NewScheme()
	for _, add := range []func(*runtime.Scheme) error{corev1.AddToScheme, namespacedapis.AddToScheme, clusterapis.AddToScheme} {
		if err := add(s); err != nil {
			t.Fatal(err)
		}
	}
	return fake.NewClientBuilder().WithScheme(s).WithObjects(objs...).Build()
}

func nsKey(kind, pc string) *nsrepo.DeployKey {
	k := &nsrepo.DeployKey{ObjectMeta: metav1.ObjectMeta{Namespace: "team", Name: "k", UID: "u1"}}
	k.SetProviderConfigReference(&xpv2.ProviderConfigReference{Kind: kind, Name: pc})
	return k
}

func TestSetupNamespaced(t *testing.T) {
	ctx := context.Background()
	cpc := &namespacedv1beta1.ClusterProviderConfig{ObjectMeta: metav1.ObjectMeta{Name: "default"}, Spec: secretRef("crossplane-system")}
	// The team's ProviderConfig names the platform's Secret: its own
	// namespace's is read instead.
	pc := &namespacedv1beta1.ProviderConfig{ObjectMeta: metav1.ObjectMeta{Namespace: "team", Name: "ci"}, Spec: secretRef("crossplane-system")}
	k := kubeWith(t, cpc, pc,
		sec("crossplane-system", map[string]string{"credentials": `{"host": "https://forgejo", "api_token": "admin"}`}),
		sec("team", map[string]string{"host": "https://forgejo", "username": "ci-team", "password": "pw"}))
	setup := TerraformSetupBuilder()

	ps, err := setup(ctx, k, nsKey("ClusterProviderConfig", "default"))
	if err != nil || ps.Configuration["api_token"] != "admin" || ps.FrameworkProvider == nil {
		t.Fatalf("the ClusterProviderConfig's JSON: %v, token set: %t", err, ps.Configuration["api_token"] == "admin")
	}
	ps, err = setup(ctx, k, nsKey("ProviderConfig", "ci"))
	if err != nil || ps.Configuration["username"] != "ci-team" || ps.Configuration["api_token"] != nil {
		t.Fatalf("the team's own flat keys: %v, user %v", err, ps.Configuration["username"])
	}

	for _, tc := range []struct {
		name string
		mg   resource.Managed
		want string
	}{
		{"no reference", &nsrepo.DeployKey{ObjectMeta: metav1.ObjectMeta{Namespace: "team", Name: "k"}}, "no providerConfigRef"},
		{"unknown kind", nsKey("Other", "x"), "unknown GVK"},
		{"missing config", nsKey("ProviderConfig", "none"), "cannot get referenced ProviderConfig"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := setup(ctx, k, tc.mg); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q, got %v", tc.want, err)
			}
		})
	}

	noHost := &namespacedv1beta1.ProviderConfig{ObjectMeta: metav1.ObjectMeta{Namespace: "other", Name: "ci"}, Spec: secretRef("other")}
	k = kubeWith(t, noHost, sec("other", map[string]string{"api_token": "t"}))
	key := nsKey("ProviderConfig", "ci")
	key.Namespace = "other"
	if _, err := setup(ctx, k, key); err == nil || !strings.Contains(err.Error(), "no host") {
		t.Fatalf("credentials without host: %v", err)
	}
}

func TestSetupClusterScoped(t *testing.T) {
	ctx := context.Background()
	spec := clusterv1beta1.ProviderConfigSpec{}
	spec.Credentials.Source = xpv2.CredentialsSourceSecret
	spec.Credentials.SecretRef = &xpv2.SecretKeySelector{SecretReference: xpv2.SecretReference{Namespace: "crossplane-system", Name: "forgejo"}, Key: "credentials"}
	pc := &clusterv1beta1.ProviderConfig{ObjectMeta: metav1.ObjectMeta{Name: "default"}, Spec: spec}
	k := kubeWith(t, pc, sec("crossplane-system", map[string]string{"credentials": `{"host": "https://forgejo", "api_token": "admin"}`}))
	key := &clusterrepo.DeployKey{ObjectMeta: metav1.ObjectMeta{Name: "k", UID: "u2"}}
	key.SetProviderConfigReference(&xpv2.Reference{Name: "default"})
	ps, err := TerraformSetupBuilder()(ctx, k, key)
	if err != nil || ps.Configuration["host"] != "https://forgejo" {
		t.Fatalf("a cluster-scoped object's ProviderConfig: %v", err)
	}
	key.SetProviderConfigReference(nil)
	if _, err := TerraformSetupBuilder()(ctx, k, key); err == nil || !strings.Contains(err.Error(), "no providerConfigRef") {
		t.Fatalf("no reference: %v", err)
	}
	key.SetProviderConfigReference(&xpv2.Reference{Name: "none"})
	if _, err := TerraformSetupBuilder()(ctx, k, key); err == nil || !strings.Contains(err.Error(), "cannot get referenced ProviderConfig") {
		t.Fatalf("missing config: %v", err)
	}
	if s, err := toSharedPCSpec(nil); s != nil || err != nil {
		t.Fatal("no config, no spec")
	}
}
