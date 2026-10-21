// SPDX-FileCopyrightText: 2026 Vincent Brobald
// SPDX-License-Identifier: MPL-2.0

package clients

import (
	"context"
	"testing"

	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	namespacedv1beta1 "brobald.be/provider-forgejo/apis/namespaced/v1beta1"
)

// pcSpec returns a spec whose credentials are the key of the Secret
// crossplane-system/forgejo-credentials.
func pcSpec(key string) *namespacedv1beta1.ProviderConfigSpec {
	spec := &namespacedv1beta1.ProviderConfigSpec{}
	spec.Credentials.Source = xpv2.CredentialsSourceSecret
	spec.Credentials.SecretRef = &xpv2.SecretKeySelector{
		SecretReference: xpv2.SecretReference{Namespace: "crossplane-system", Name: "forgejo-credentials"},
		Key:             key,
	}
	return spec
}

// withSecret returns a fake client builder holding that Secret with data.
func withSecret(data map[string]string) *fake.ClientBuilder {
	s := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: "crossplane-system", Name: "forgejo-credentials"},
		Data: map[string][]byte{}}
	for k, v := range data {
		s.Data[k] = []byte(v)
	}
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(s)
}

// TestCredentialsJSON reads the credentials as JSON under the selected key.
func TestCredentialsJSON(t *testing.T) {
	c := withSecret(map[string]string{"credentials": `{"host":"https://git.example","api_token":"t0k"}`}).Build()
	creds, err := extractCredentials(context.Background(), c, pcSpec("credentials"))
	if err != nil {
		t.Fatal(err)
	}
	if creds["host"] != "https://git.example" || creds["api_token"] != "t0k" {
		t.Fatalf("got %v", creds)
	}
}

// TestCredentialsFlatKeys reads flat keys from a Secret without the selected
// key, ignoring the others.
func TestCredentialsFlatKeys(t *testing.T) {
	c := withSecret(map[string]string{"host": "https://git.example", "api_token": "t0k", "other": "x"}).Build()
	creds, err := extractCredentials(context.Background(), c, pcSpec("credentials"))
	if err != nil {
		t.Fatal(err)
	}
	if len(creds) != 2 || creds["host"] != "https://git.example" || creds["api_token"] != "t0k" {
		t.Fatalf("got %v", creds)
	}
}

// TestCredentialsBadJSON checks that malformed JSON is an error.
func TestCredentialsBadJSON(t *testing.T) {
	c := withSecret(map[string]string{"credentials": "not json"}).Build()
	if _, err := extractCredentials(context.Background(), c, pcSpec("credentials")); err == nil {
		t.Fatal("malformed JSON must be an error")
	}
}
