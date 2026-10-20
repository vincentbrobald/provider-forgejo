// SPDX-FileCopyrightText: 2024 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/upjet/v2/pkg/controller"

	organization "brobald.be/provider-forgejo/internal/controller/cluster/organization/organization"
	organizationowner "brobald.be/provider-forgejo/internal/controller/cluster/organization/organizationowner"
	team "brobald.be/provider-forgejo/internal/controller/cluster/organization/team"
	teammember "brobald.be/provider-forgejo/internal/controller/cluster/organization/teammember"
	providerconfig "brobald.be/provider-forgejo/internal/controller/cluster/providerconfig"
	branchprotection "brobald.be/provider-forgejo/internal/controller/cluster/repository/branchprotection"
	collaborator "brobald.be/provider-forgejo/internal/controller/cluster/repository/collaborator"
	deploykey "brobald.be/provider-forgejo/internal/controller/cluster/repository/deploykey"
	repository "brobald.be/provider-forgejo/internal/controller/cluster/repository/repository"
	teamrepository "brobald.be/provider-forgejo/internal/controller/cluster/repository/teamrepository"
	webhook "brobald.be/provider-forgejo/internal/controller/cluster/repository/webhook"
	personalaccesstoken "brobald.be/provider-forgejo/internal/controller/cluster/user/personalaccesstoken"
	user "brobald.be/provider-forgejo/internal/controller/cluster/user/user"
)

// Setup creates all controllers with the supplied logger and adds them to
// the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		organization.Setup,
		organizationowner.Setup,
		team.Setup,
		teammember.Setup,
		providerconfig.Setup,
		branchprotection.Setup,
		collaborator.Setup,
		deploykey.Setup,
		repository.Setup,
		teamrepository.Setup,
		webhook.Setup,
		personalaccesstoken.Setup,
		user.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}

// SetupGated creates all controllers with the supplied logger and adds them to
// the supplied manager gated.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		organization.SetupGated,
		organizationowner.SetupGated,
		team.SetupGated,
		teammember.SetupGated,
		providerconfig.SetupGated,
		branchprotection.SetupGated,
		collaborator.SetupGated,
		deploykey.SetupGated,
		repository.SetupGated,
		teamrepository.SetupGated,
		webhook.SetupGated,
		personalaccesstoken.SetupGated,
		user.SetupGated,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}

// SetupWebhookWithManager registers conversion webhooks for all resource kinds in the group.
func SetupWebhookWithManager(mgr ctrl.Manager) error {
	for _, setup := range []func(ctrl.Manager) error{
		organization.SetupWebhookWithManager,
		organizationowner.SetupWebhookWithManager,
		team.SetupWebhookWithManager,
		teammember.SetupWebhookWithManager,
		providerconfig.SetupWebhookWithManager,
		branchprotection.SetupWebhookWithManager,
		collaborator.SetupWebhookWithManager,
		deploykey.SetupWebhookWithManager,
		repository.SetupWebhookWithManager,
		teamrepository.SetupWebhookWithManager,
		webhook.SetupWebhookWithManager,
		personalaccesstoken.SetupWebhookWithManager,
		user.SetupWebhookWithManager,
	} {
		if err := setup(mgr); err != nil {
			return err
		}
	}
	return nil
}
