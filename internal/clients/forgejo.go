// SPDX-FileCopyrightText: 2024 The Crossplane Authors <https://crossplane.io>
// SPDX-FileCopyrightText: 2026 Vincent Brobald
//
// SPDX-License-Identifier: Apache-2.0
//
// From crossplane/upjet-provider-template, changed for provider-forgejo.

// Package clients turns a ProviderConfig into the configuration of the
// Forgejo Terraform provider, which runs in this process (Plugin Framework).
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

// credentialKeys are the credentials the Terraform provider takes: host
// (https://forgejo.example), and either api_token or username and password.
// The server's certificate is checked against the system CAs (SSL_CERT_DIR
// adds others).
var credentialKeys = []string{"host", "api_token", "username", "password"}

// TerraformSetupBuilder returns the function that configures the Terraform
// provider for a managed resource, from its ProviderConfig's credentials.
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

// extractCredentials reads the credentials, in either form:
//   - JSON in the Secret key the ProviderConfig names ({"host": ..., "api_token": ...});
//   - flat keys (host, api_token, ...) when the Secret has no such key, so
//     that each value can live in its own key (e.g. a token kept in sync by
//     another tool).
//
// Sources other than a Secret go through Crossplane's extractor (JSON).
func extractCredentials(ctx context.Context, c client.Client, pcSpec *namespacedv1beta1.ProviderConfigSpec) (map[string]string, error) {
	creds := map[string]string{}
	sel := pcSpec.Credentials.CommonCredentialSelectors
	if pcSpec.Credentials.Source == xpv2.CredentialsSourceSecret && sel.SecretRef != nil {
		secret := &corev1.Secret{}
		if err := c.Get(ctx, types.NamespacedName{Namespace: sel.SecretRef.Namespace, Name: sel.SecretRef.Name}, secret); err != nil {
			return nil, errors.Wrap(err, errExtractCredentials)
		}
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

func resolveModern(ctx context.Context, crClient client.Client, mg resource.ModernManaged) (*namespacedv1beta1.ProviderConfigSpec, error) {
	configRef := mg.GetProviderConfigReference()
	if configRef == nil {
		return nil, errors.New(errNoProviderConfig)
	}

	pcRuntimeObj, err := crClient.Scheme().New(namespacedv1beta1.SchemeGroupVersion.WithKind(configRef.Kind))
	if err != nil {
		return nil, errors.Wrap(err, "unknown GVK for ProviderConfig")
	}
	pcObj, ok := pcRuntimeObj.(client.Object)
	if !ok {
		return nil, errors.New(" is not an Object")
	}

	// Namespace is ignored for the cluster-scoped ClusterProviderConfig.
	if err := crClient.Get(ctx, types.NamespacedName{Name: configRef.Name, Namespace: mg.GetNamespace()}, pcObj); err != nil {
		return nil, errors.Wrap(err, errGetProviderConfig)
	}

	var pcSpec namespacedv1beta1.ProviderConfigSpec
	pcu := &namespacedv1beta1.ProviderConfigUsage{}
	switch pc := pcObj.(type) {
	case *namespacedv1beta1.ProviderConfig:
		pcSpec = pc.Spec
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
