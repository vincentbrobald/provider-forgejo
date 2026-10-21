// SPDX-FileCopyrightText: 2024 The Crossplane Authors <https://crossplane.io>
// SPDX-FileCopyrightText: 2026 Vincent Brobald
//
// SPDX-License-Identifier: Apache-2.0
//
// From crossplane/upjet-provider-template, changed for provider-forgejo.

// Package clients gives Upjet the Terraform provider configuration of a
// managed resource, from its ProviderConfig.
package clients

import (
	"context"
	"encoding/json"

	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/upjet/v2/pkg/terraform"

	clusterv1beta1 "brobald.be/provider-forgejo/apis/cluster/v1beta1"
	namespacedv1beta1 "brobald.be/provider-forgejo/apis/namespaced/v1beta1"
	tfprovider "brobald.be/provider-forgejo/internal/tfprovider"
	"brobald.be/provider-forgejo/internal/tfversion"
)

const (
	errNoProviderConfig     = "no providerConfigRef provided"
	errGetProviderConfig    = "cannot get referenced ProviderConfig"
	errTrackUsage           = "cannot track ProviderConfig usage"
	errExtractCredentials   = "cannot extract credentials"
	errUnmarshalCredentials = "cannot unmarshal forgejo credentials as JSON"
)

// credentialKeys are the credential fields passed to the Terraform provider
// configuration, and the keys read from a Secret holding flat keys.
var credentialKeys = []string{"host", "api_token", "username", "password"}

// TerraformSetupBuilder returns the function Upjet calls whenever it connects
// to a managed resource: it resolves the resource's ProviderConfig (recording
// the usage), reads its credentials, and returns them as the Terraform
// provider configuration, with a new instance of the provider compiled in.
// Credentials without a host are an error.
func TerraformSetupBuilder() terraform.SetupFn {
	return func(ctx context.Context, client client.Client, mg resource.Managed) (terraform.Setup, error) {
		ps := terraform.Setup{}

		pcSpec, err := resolveProviderConfig(ctx, client, mg)
		if err != nil {
			return terraform.Setup{}, errors.Wrap(err, "cannot resolve provider config")
		}

		creds, err := extractCredentials(ctx, client, pcSpec)
		if err != nil {
			return ps, err
		}
		if creds["host"] == "" {
			return ps, errors.New("the credentials have no host")
		}

		// Only the known fields, and only those present: an absent field
		// stays unset rather than empty.
		ps.Configuration = map[string]any{}
		for _, k := range credentialKeys {
			if v, ok := creds[k]; ok {
				ps.Configuration[k] = v
			}
		}
		ps.FrameworkProvider = tfprovider.New(tfversion.Version)()
		return ps, nil
	}
}

// extractCredentials returns the credential fields of a ProviderConfig spec.
// From a Secret without the selected key, it takes the keys of
// credentialKeys (others are ignored); otherwise it reads the data of the
// source (the Secret's key, an environment variable or a file) as a JSON
// object of strings. The sources None and InjectedIdentity fail.
func extractCredentials(ctx context.Context, c client.Client, pcSpec *namespacedv1beta1.ProviderConfigSpec) (map[string]string, error) {
	creds := map[string]string{}
	sel := pcSpec.Credentials.CommonCredentialSelectors
	if pcSpec.Credentials.Source == xpv2.CredentialsSourceSecret && sel.SecretRef != nil {
		secret := &corev1.Secret{}
		if err := c.Get(ctx, types.NamespacedName{Namespace: sel.SecretRef.Namespace, Name: sel.SecretRef.Name}, secret); err != nil {
			return nil, errors.Wrap(err, errExtractCredentials)
		}
		// No such key: flat keys. With the key, fall through to the common
		// extractor, which reads the Secret again.
		if _, ok := secret.Data[sel.SecretRef.Key]; !ok {
			for _, k := range credentialKeys {
				if v, ok := secret.Data[k]; ok {
					creds[k] = string(v)
				}
			}
			return creds, nil
		}
	}
	data, err := resource.CommonCredentialExtractor(ctx, pcSpec.Credentials.Source, c, sel)
	if err != nil {
		return nil, errors.Wrap(err, errExtractCredentials)
	}
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, errors.Wrap(err, errUnmarshalCredentials)
	}
	return creds, nil
}

// toSharedPCSpec converts a cluster-scoped ProviderConfig's spec to the
// namespaced spec type (same fields) by a JSON round trip, so the rest handles
// one type. A nil ProviderConfig gives nil and no error.
func toSharedPCSpec(pc *clusterv1beta1.ProviderConfig) (*namespacedv1beta1.ProviderConfigSpec, error) {
	if pc == nil {
		return nil, nil
	}
	data, err := json.Marshal(pc.Spec)
	if err != nil {
		return nil, err
	}

	var mSpec namespacedv1beta1.ProviderConfigSpec
	err = json.Unmarshal(data, &mSpec)
	return &mSpec, err
}

// resolveProviderConfig returns the spec of the ProviderConfig a managed
// resource references and records that the resource uses it. Cluster-scoped
// (legacy) resources reference a cluster-scoped ProviderConfig; namespaced
// (modern) ones a ProviderConfig of their namespace or a
// ClusterProviderConfig.
func resolveProviderConfig(ctx context.Context, crClient client.Client, mg resource.Managed) (*namespacedv1beta1.ProviderConfigSpec, error) {
	switch managed := mg.(type) {
	case resource.LegacyManaged: //nolint:staticcheck // still handling cluster-scoped behavior
		return resolveLegacy(ctx, crClient, managed)
	case resource.ModernManaged:
		return resolveModern(ctx, crClient, managed)
	default:
		return nil, errors.New("resource is not a managed resource")
	}
}

// resolveLegacy resolves the cluster-scoped ProviderConfig of a cluster-scoped
// managed resource and records a cluster-scoped ProviderConfigUsage.
func resolveLegacy(ctx context.Context, client client.Client, mg resource.LegacyManaged) (*namespacedv1beta1.ProviderConfigSpec, error) { //nolint:staticcheck // still handling cluster-scoped behavior
	configRef := mg.GetProviderConfigReference()
	if configRef == nil {
		return nil, errors.New(errNoProviderConfig)
	}
	pc := &clusterv1beta1.ProviderConfig{}
	if err := client.Get(ctx, types.NamespacedName{Name: configRef.Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetProviderConfig)
	}

	t := resource.NewLegacyProviderConfigUsageTracker(client, &clusterv1beta1.ProviderConfigUsage{})
	if err := t.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackUsage)
	}

	return toSharedPCSpec(pc)
}

// resolveModern resolves the ProviderConfig or ClusterProviderConfig of a
// namespaced managed resource and records a ProviderConfigUsage. An unknown
// kind in the reference is an error.
func resolveModern(ctx context.Context, crClient client.Client, mg resource.ModernManaged) (*namespacedv1beta1.ProviderConfigSpec, error) {
	configRef := mg.GetProviderConfigReference()
	if configRef == nil {
		return nil, errors.New(errNoProviderConfig)
	}

	// The reference's kind picks the type; only a ProviderConfig is looked up
	// in the resource's namespace.
	pcRuntimeObj, err := crClient.Scheme().New(namespacedv1beta1.SchemeGroupVersion.WithKind(configRef.Kind))
	if err != nil {
		return nil, errors.Wrap(err, "unknown GVK for ProviderConfig")
	}
	pcObj, ok := pcRuntimeObj.(client.Object)
	if !ok {
		return nil, errors.Errorf("%s is not an Object", configRef.Kind)
	}

	key := types.NamespacedName{Name: configRef.Name}
	if configRef.Kind == namespacedv1beta1.ProviderConfigKind {
		key.Namespace = mg.GetNamespace()
	}
	if err := crClient.Get(ctx, key, pcObj); err != nil {
		return nil, errors.Wrap(err, errGetProviderConfig)
	}

	var pcSpec namespacedv1beta1.ProviderConfigSpec
	pcu := &namespacedv1beta1.ProviderConfigUsage{}
	switch pc := pcObj.(type) {
	case *namespacedv1beta1.ProviderConfig:
		pcSpec = pc.Spec
		// A namespaced ProviderConfig's Secret is always read from its own
		// namespace, whatever the reference says: a team's ProviderConfig
		// can't point at another namespace's credentials.
		if pcSpec.Credentials.SecretRef != nil {
			pcSpec.Credentials.SecretRef.Namespace = mg.GetNamespace()
		}
	case *namespacedv1beta1.ClusterProviderConfig:
		pcSpec = pc.Spec
	default:
		return nil, errors.New("unknown provider config type")
	}
	t := resource.NewProviderConfigUsageTracker(crClient, pcu)
	if err := t.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackUsage)
	}
	return &pcSpec, nil
}
