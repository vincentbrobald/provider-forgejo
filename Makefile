# provider-forgejo: Crossplane provider for Forgejo, generated with Upjet from
# the Forgejo Terraform provider (svalabs, vendored in third_party/).
#
# Needs Go; `make podman-<target>` runs <target> in the pinned Go image
# instead (e.g. make podman-generate).
#
#   make vendor     re-vendor the Terraform provider (hack/vendor-tf-provider.sh)
#   make schema     config/schema.json, from the vendored provider (OpenTofu)
#   make generate   CRDs, Go types, controllers, examples (Upjet)
#   make test       unit tests
#   make build      bin/provider
#   make package-yaml  bin/package.yaml: crossplane.yaml and the CRDs, the
#                   package's contents (the CI's image build adds it)
#   make image      the runtime image (podman), saved as .build/image.tar
#   make xpkg       the Crossplane package .build/provider-forgejo.xpkg
#                   (Crossplane CLI, built from source)

GO_IMAGE      ?= docker.io/library/golang:1.26.9
# OpenTofu only asks the vendored provider for its schema at generation time;
# nothing runs Terraform or OpenTofu at runtime.
TOFU_VERSION  ?= 1.13.1
TOFU_SHA256   ?= $(shell cat hack/tofu.sha256 2>/dev/null)
WORK          := $(CURDIR)/.work
TOFU          := $(WORK)/tofu-$(TOFU_VERSION)/tofu
CROSSPLANE_CLI_VERSION ?= v2.5.0
CROSSPLANE    := $(WORK)/bin/crossplane-$(CROSSPLANE_CLI_VERSION)
VERSION       ?= $(shell cat VERSION)

.PHONY: vendor schema generate test build package-yaml image xpkg podman-%

vendor:
	hack/vendor-tf-provider.sh

$(TOFU):
	@mkdir -p $(WORK)/tofu-$(TOFU_VERSION)
	curl -fsSL -o $(WORK)/tofu.tar.gz https://github.com/opentofu/opentofu/releases/download/v$(TOFU_VERSION)/tofu_$(TOFU_VERSION)_linux_amd64.tar.gz
	echo "$(TOFU_SHA256)  $(WORK)/tofu.tar.gz" | sha256sum -c --quiet -
	tar xzf $(WORK)/tofu.tar.gz -C $(WORK)/tofu-$(TOFU_VERSION) tofu
	rm -f $(WORK)/tofu.tar.gz

schema: $(TOFU)
	@mkdir -p $(WORK)/schema/plugins
	go build -o $(WORK)/schema/plugins/terraform-provider-forgejo ./hack/tfprovider
	printf 'provider_installation {\n  dev_overrides {\n    "svalabs/forgejo" = "%s"\n  }\n  direct {}\n}\n' $(WORK)/schema/plugins > $(WORK)/schema/tofurc
	printf 'terraform {\n  required_providers {\n    forgejo = { source = "svalabs/forgejo" }\n  }\n}\n' > $(WORK)/schema/main.tf
	cd $(WORK)/schema && TF_CLI_CONFIG_FILE=$(WORK)/schema/tofurc $(TOFU) providers schema -json > $(CURDIR)/config/schema.json

# Upjet runs goimports on what it writes (version: go.mod's golang.org/x/tools).
generate:
	@mkdir -p internal/controller
	go install golang.org/x/tools/cmd/goimports
	PATH="$$(go env GOPATH)/bin:$$PATH" go generate ./apis/...

test:
	go test ./...

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X $(shell go list -m)/internal/version.Version=$(VERSION)" \
	  -o bin/provider ./cmd/provider

package-yaml:
	@mkdir -p bin
	{ cat package/crossplane.yaml; for f in package/crds/*.yaml; do echo '---'; cat "$$f"; done; } > bin/package.yaml

image:
	@mkdir -p .build
	podman build --target runtime -t provider-forgejo:$(VERSION) --build-arg VERSION=$(VERSION) .
	podman save -q --format docker-archive -o .build/image.tar provider-forgejo:$(VERSION)

# The CLI isn't published as a binary for v2: built from its module
# (checksums from Go's sum database).
$(CROSSPLANE):
	@mkdir -p $(WORK)/bin
	GOBIN=$(WORK)/bin go install github.com/crossplane/cli/v2/cmd/crossplane@$(CROSSPLANE_CLI_VERSION)
	mv $(WORK)/bin/crossplane $(CROSSPLANE)

xpkg: $(CROSSPLANE)
	$(CROSSPLANE) xpkg build --package-root=package --embed-runtime-image-tarball=.build/image.tar \
	  --package-file=.build/provider-forgejo.xpkg

podman-%:
	podman run --rm -v $(CURDIR):/src:Z -v provider-forgejo-gocache:/go -w /src $(GO_IMAGE) make $*
