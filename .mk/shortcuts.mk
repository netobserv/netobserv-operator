##@ shortcuts helpers
.PHONY: tls-scan
tls-scan: tls-scanner ## Alias for tls-scanner.

.PHONY: tls-scanner-image
tls-scanner-image: tls-scanner-image-build ## Build and push the tls-scanner image.
	$(MAKE) tls-scanner-image-push

.PHONY: build-image
build-image: image-build ## Build MULTIARCH_TARGETS images

.PHONY: push-image
push-image: image-push ## Push MULTIARCH_TARGETS images

.PHONY: build-manifest
build-manifest: manifest-build ## Build MULTIARCH_TARGETS manifest

.PHONY: push-manifest
push-manifest: manifest-push ## Push MULTIARCH_TARGETS manifest

.PHONY: images
images: image-build image-push manifest-build manifest-push ## Build and push MULTIARCH_TARGETS images and related manifest

.PHONY: build-catalog
build-catalog: catalog-build ## Build a catalog image

.PHONY: push-catalog
push-catalog: catalog-push ## Push a catalog image

.PHONY: catalog
catalog: catalog-build catalog-push ## Build and push a catalog image

.PHONY: build-bundle
build-bundle: bundle-build ## Build the bundle image

.PHONY: push-bundle
push-bundle: bundle-push ## Push the bundle image

.PHONY: bundle-all
bundle-all: bundle bundle-build bundle-push ## Build and push the bundle image
