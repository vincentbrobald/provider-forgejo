// SPDX-FileCopyrightText: 2024 The Crossplane Authors <https://crossplane.io>
// SPDX-FileCopyrightText: 2026 Vincent Brobald
//
// SPDX-License-Identifier: Apache-2.0
//
// From crossplane/upjet-provider-template, changed for provider-forgejo.

// Command provider runs the Crossplane provider for Forgejo: the controllers
// of its managed resources and ProviderConfigs, with the Forgejo Terraform
// provider compiled in.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/alecthomas/kingpin/v2"
	changelogsv1alpha1 "github.com/crossplane/crossplane-runtime/v2/apis/changelogs/proto/v1alpha1"
	xpcontroller "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/gate"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/customresourcesgate"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"
	tjcontroller "github.com/crossplane/upjet/v2/pkg/controller"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	apisCluster "brobald.be/provider-forgejo/apis/cluster"
	apisNamespaced "brobald.be/provider-forgejo/apis/namespaced"
	"brobald.be/provider-forgejo/config"
	"brobald.be/provider-forgejo/internal/clients"
	controllerCluster "brobald.be/provider-forgejo/internal/controller/cluster"
	controllerNamespaced "brobald.be/provider-forgejo/internal/controller/namespaced"
	"brobald.be/provider-forgejo/internal/features"
	"brobald.be/provider-forgejo/internal/version"
)

// The environment variables that may name the directory of the server key
// and certificate, and the default directory.
const (
	webhookTLSCertDirEnvVar = "WEBHOOK_TLS_CERT_DIR"
	tlsServerCertDirEnvVar  = "TLS_SERVER_CERTS_DIR"
	certsDirEnvVar          = "CERTS_DIR"
	tlsServerCertDir        = "/tls/server"
)

// main parses the flags (each also settable through an environment variable,
// see DefaultEnvars), sets up a controller manager with the controllers of
// both scopes, and runs it until a termination signal. Any setup error ends
// the process.
func main() {
	var (
		app                     = kingpin.New(filepath.Base(os.Args[0]), "Crossplane provider for Forgejo (Upjet, Forgejo Terraform provider compiled in)").DefaultEnvars()
		debug                   = app.Flag("debug", "Run with debug logging.").Short('d').Bool()
		syncPeriod              = app.Flag("sync", "Controller manager sync period such as 300ms, 1.5h, or 2h45m").Short('s').Default("1h").Duration()
		pollInterval            = app.Flag("poll", "Poll interval controls how often an individual resource should be checked for drift.").Default("10m").Duration()
		pollStateMetricInterval = app.Flag("poll-state-metric", "State metric recording interval").Default("5s").Duration()
		leaderElection          = app.Flag("leader-election", "Use leader election for the controller manager.").Short('l').Default("false").OverrideDefaultFromEnvar("LEADER_ELECTION").Bool()
		maxReconcileRate        = app.Flag("max-reconcile-rate", "The global maximum rate per second at which resources may be checked for drift from the desired state.").Default("10").Int()

		webhookPort          = app.Flag("webhook-port", "The port the webhook listens on").Default("9443").Envar("WEBHOOK_PORT").Int()
		metricsBindAddress   = app.Flag("metrics-bind-address", "The address the metrics server listens on").Default(":8080").Envar("METRICS_BIND_ADDRESS").String()
		changelogsSocketPath = app.Flag("changelogs-socket-path", "Path for changelogs socket (if enabled)").Default("/var/run/changelogs/changelogs.sock").Envar("CHANGELOGS_SOCKET_PATH").String()

		enableManagementPolicies = app.Flag("enable-management-policies", "Enable support for Management Policies.").Default("true").Envar("ENABLE_MANAGEMENT_POLICIES").Bool()
		enableChangeLogs         = app.Flag("enable-changelogs", "Enable support for capturing change logs during reconciliation.").Default("false").Envar("ENABLE_CHANGE_LOGS").Bool()
		enableSecretCache        = app.Flag("enable-secret-cache", "Enable caching of Secret objects. When true, Secrets are served from the informer cache instead of direct API calls. This reduces API server load but increases memory usage.").Default("true").Envar("ENABLE_SECRET_CACHE").Bool()

		// certsDirSet tells whether --certs-dir was given.
		certsDirSet = false
		certsDir    = app.Flag("certs-dir", "The directory that contains the server key and certificate.").Default(tlsServerCertDir).Envar(certsDirEnvVar).PreAction(func(_ *kingpin.ParseContext) error {
			certsDirSet = true
			return nil
		}).String()
	)

	kingpin.MustParse(app.Parse(os.Args[1:]))

	zl := zap.New(zap.UseDevMode(*debug))
	log := logging.NewLogrLogger(zl.WithName("provider-forgejo"))
	if *debug {
		ctrl.SetLogger(zl)
	}

	log.Debug("Starting", "sync-period", syncPeriod.String(), "poll-interval", pollInterval.String(), "max-reconcile-rate", *maxReconcileRate)

	cfg, err := ctrl.GetConfig()
	kingpin.FatalIfError(err, "Cannot get API server rest config")

	// Unless --certs-dir was given, take the directory from CERTS_DIR, else
	// TLS_SERVER_CERTS_DIR (newer Crossplane versions), else
	// WEBHOOK_TLS_CERT_DIR (older ones); without any, keep the default.
	if !certsDirSet {
		xpCertsDir := os.Getenv(certsDirEnvVar)
		if xpCertsDir == "" {
			xpCertsDir = os.Getenv(tlsServerCertDirEnvVar)
		}
		if xpCertsDir == "" {
			xpCertsDir = os.Getenv(webhookTLSCertDirEnvVar)
		}
		if xpCertsDir != "" {
			*certsDir = xpCertsDir
		}
	}

	// With the Secret cache off, the client reads Secrets from the API server.
	var clientOpts client.Options
	if !*enableSecretCache {
		clientOpts = client.Options{
			Cache: &client.CacheOptions{
				DisableFor: []client.Object{&corev1.Secret{}},
			},
		}
	}

	scheme := runtime.NewScheme()
	kingpin.FatalIfError(clientgoscheme.AddToScheme(scheme), "Cannot add clientgo scheme")
	kingpin.FatalIfError(apisCluster.AddToScheme(scheme), "Cannot add cluster-scoped Forgejo APIs to scheme")
	kingpin.FatalIfError(apisNamespaced.AddToScheme(scheme), "Cannot add namespace-scoped Forgejo APIs to scheme")
	kingpin.FatalIfError(apiextensionsv1.AddToScheme(scheme), "Cannot add CustomResourceDefinition to scheme")
	kingpin.FatalIfError(authv1.AddToScheme(scheme), "Cannot add k8s authorization APIs to scheme")

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:           scheme,
		LeaderElection:   *leaderElection,
		LeaderElectionID: "crossplane-leader-election-provider-forgejo",
		Cache: cache.Options{
			SyncPeriod: syncPeriod,
			// CRDs are cached without their schemas: the CRD gate needs
			// little more than their names and status.
			ByObject: map[client.Object]cache.ByObject{
				&apiextensionsv1.CustomResourceDefinition{}: {
					Transform: customresourcesgate.TransformStripCRDSchema,
				},
			},
		},
		Client: clientOpts,
		Metrics: metricsserver.Options{
			BindAddress: *metricsBindAddress,
		},
		WebhookServer: webhook.NewServer(
			webhook.Options{
				CertDir: *certsDir,
				Port:    *webhookPort,
			}),
		LeaderElectionResourceLock: resourcelock.LeasesResourceLock,
		LeaseDuration:              func() *time.Duration { d := 60 * time.Second; return &d }(),
		RenewDeadline:              func() *time.Duration { d := 50 * time.Second; return &d }(),
	})
	kingpin.FatalIfError(err, "Cannot create controller manager")

	metricRecorder := managed.NewMRMetricRecorder()
	stateMetrics := statemetrics.NewMRStateMetrics()

	metrics.Registry.MustRegister(metricRecorder)
	metrics.Registry.MustRegister(stateMetrics)

	// The controller options of the cluster-scoped kinds and of the namespaced
	// ones differ only in the Upjet provider configuration. The global rate
	// limit (--max-reconcile-rate) also bounds the concurrent reconciles of
	// each controller.
	clusterOpts := tjcontroller.Options{
		Options: xpcontroller.Options{
			Logger:                  log,
			GlobalRateLimiter:       ratelimiter.NewGlobal(*maxReconcileRate),
			PollInterval:            *pollInterval,
			MaxConcurrentReconciles: *maxReconcileRate,
			Features:                &feature.Flags{},
			MetricOptions: &xpcontroller.MetricOptions{
				PollStateMetricInterval: *pollStateMetricInterval,
				MRMetrics:               metricRecorder,
				MRStateMetrics:          stateMetrics,
			},
		},
		Provider: config.GetProvider(),
		// The Terraform provider runs in this process (no Terraform CLI):
		// Upjet keeps each resource's state in its operation tracker.
		OperationTrackerStore: tjcontroller.NewOperationStore(log),
		SetupFn:               clients.TerraformSetupBuilder(),
		StartWebhooks:         *certsDir != "",
	}

	namespacedOpts := tjcontroller.Options{
		Options: xpcontroller.Options{
			Logger:                  log,
			GlobalRateLimiter:       ratelimiter.NewGlobal(*maxReconcileRate),
			PollInterval:            *pollInterval,
			MaxConcurrentReconciles: *maxReconcileRate,
			Features:                &feature.Flags{},
			MetricOptions: &xpcontroller.MetricOptions{
				PollStateMetricInterval: *pollStateMetricInterval,
				MRMetrics:               metricRecorder,
				MRStateMetrics:          stateMetrics,
			},
		},
		Provider: config.GetProviderNamespaced(),
		// The Terraform provider runs in this process (no Terraform CLI):
		// Upjet keeps each resource's state in its operation tracker.
		OperationTrackerStore: tjcontroller.NewOperationStore(log),
		SetupFn:               clients.TerraformSetupBuilder(),
		StartWebhooks:         *certsDir != "",
	}

	if *enableManagementPolicies {
		clusterOpts.Features.Enable(features.EnableBetaManagementPolicies)
		namespacedOpts.Features.Enable(features.EnableBetaManagementPolicies)
		log.Info("Beta feature enabled", "flag", features.EnableBetaManagementPolicies)
	}

	if *enableChangeLogs {
		clusterOpts.Features.Enable(feature.EnableAlphaChangeLogs)
		namespacedOpts.Features.Enable(feature.EnableAlphaChangeLogs)
		log.Info("Alpha feature enabled", "flag", feature.EnableAlphaChangeLogs)

		conn, err := grpc.NewClient("unix://"+*changelogsSocketPath, grpc.WithTransportCredentials(insecure.NewCredentials()))
		kingpin.FatalIfError(err, "failed to create change logs client connection at %s", *changelogsSocketPath)

		clo := xpcontroller.ChangeLogOptions{
			ChangeLogger: managed.NewGRPCChangeLogger(
				changelogsv1alpha1.NewChangeLogServiceClient(conn),
				managed.WithProviderVersion(fmt.Sprintf("provider-forgejo:%s", version.Version))),
		}
		clusterOpts.ChangeLogOptions = &clo
		namespacedOpts.ChangeLogOptions = &clo
	}

	// SafeStart: when the provider may watch CRDs, each controller starts only
	// once the CRDs of its kinds are ready (the CRD gate calls its setup);
	// otherwise every controller starts right away.
	canSafeStart, err := canWatchCRD(context.TODO(), mgr)
	kingpin.FatalIfError(err, "SafeStart precheck failed")
	if canSafeStart {
		crdGate := new(gate.Gate[schema.GroupVersionKind])
		clusterOpts.Gate = crdGate
		namespacedOpts.Gate = crdGate
		kingpin.FatalIfError(customresourcesgate.Setup(mgr, xpcontroller.Options{
			Logger:                  log,
			Gate:                    crdGate,
			MaxConcurrentReconciles: 1,
		}), "Cannot setup CRD gate")
		kingpin.FatalIfError(controllerCluster.SetupGated(mgr, clusterOpts), "Cannot setup cluster-scoped Forgejo controllers")
		kingpin.FatalIfError(controllerNamespaced.SetupGated(mgr, namespacedOpts), "Cannot setup namespaced Forgejo controllers")
	} else {
		log.Info("Provider has missing RBAC permissions for watching CRDs, controller SafeStart capability will be disabled")
		kingpin.FatalIfError(controllerCluster.Setup(mgr, clusterOpts), "Cannot setup cluster-scoped Forgejo controllers")
		kingpin.FatalIfError(controllerNamespaced.Setup(mgr, namespacedOpts), "Cannot setup namespaced Forgejo controllers")
	}

	kingpin.FatalIfError(mgr.Start(ctrl.SetupSignalHandler()), "Cannot start controller manager")
}

// canWatchCRD reports whether the provider may get, list and watch
// CustomResourceDefinitions, by a SelfSubjectAccessReview per verb; it stops
// at the first verb denied. The manager's scheme must hold the authorization
// API (main adds it). An error means a review couldn't be made.
func canWatchCRD(ctx context.Context, mgr manager.Manager) (bool, error) {
	verbs := []string{"get", "list", "watch"}
	for _, verb := range verbs {
		sar := &authv1.SelfSubjectAccessReview{
			Spec: authv1.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &authv1.ResourceAttributes{
					Group:    "apiextensions.k8s.io",
					Resource: "customresourcedefinitions",
					Verb:     verb,
				},
			},
		}
		if err := mgr.GetClient().Create(ctx, sar); err != nil {
			return false, errors.Wrapf(err, "unable to perform RBAC check for verb %s on CustomResourceDefinitions", verb)
		}
		if !sar.Status.Allowed {
			return false, nil
		}
	}
	return true, nil
}
