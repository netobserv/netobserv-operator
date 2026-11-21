// Package e2eframework boots a disposable Kind cluster and installs the
// NetObserv operator on it via Helm, so that test packages under this module
// can exercise the operator end-to-end the way a real user would.
package e2eframework

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/support/kind"
	"sigs.k8s.io/e2e-framework/third_party/helm"
)

const (
	// ClusterName is the name of the Kind cluster created for these tests.
	ClusterName = "netobserv-e2e"
	// Namespace is where the operator and its managed components are installed.
	Namespace = "netobserv"

	// imageOrg/operatorVersion identify the locally-built operator image once
	// loaded into Kind. Nothing is ever pushed to this registry: the image is
	// side-loaded into the cluster and imagePullPolicy is patched to
	// IfNotPresent (see installOperator) so the kubelet never reaches out.
	imageOrg        = "netobserv-e2e"
	operatorVersion = "e2e"
	operatorImage   = "quay.io/" + imageOrg + "/network-observability-operator"
)

// Main boots a Kind cluster, builds the operator image from source and loads
// it into the cluster, installs cert-manager/trust-manager (required by the
// Helm chart) and the NetObserv Helm chart itself, then runs the tests in m.
func Main(m *testing.M, testEnv *env.Environment) {
	*testEnv = env.New()

	(*testEnv).Setup(
		envfuncs.CreateCluster(kind.NewProvider(), ClusterName),
		buildAndLoadOperatorImage(),
		installCertManager(),
		installOperator(),
	)
	(*testEnv).Finish(
		envfuncs.DestroyCluster(ClusterName),
	)
	os.Exit((*testEnv).Run(m))
}

func repoRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse --show-toplevel: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func run(dir string, extraEnv []string, name string, args ...string) error {
	fmt.Printf("====> %s %s\n", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), extraEnv...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

func buildAndLoadOperatorImage() env.Func {
	return func(ctx context.Context, _ *envconf.Config) (context.Context, error) {
		root, err := repoRoot()
		if err != nil {
			return ctx, err
		}
		// Build via `make tar-image` and side-load the resulting archive, rather
		// than tagging and loading out of a local image store: the Makefile
		// builds with docker or podman depending on $OCI_BIN, and
		// `kind load docker-image` only looks in docker's store. An image
		// archive is runtime-agnostic, so this works either way.
		// `tar-image` already pins MULTIARCH_TARGETS to amd64.
		buildEnv := []string{"IMAGE_ORG=" + imageOrg, "VERSION=" + operatorVersion}
		if err := run(root, buildEnv, "make", "tar-image"); err != nil {
			return ctx, err
		}
		archive := filepath.Join(root, "out", "operator.tar")
		if err := run(root, nil, "kind", "load", "image-archive", archive, "--name", ClusterName); err != nil {
			return ctx, err
		}
		return ctx, nil
	}
}

func installCertManager() env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		h := helm.New(cfg.KubeconfigFile())
		if err := h.RunRepo(helm.WithArgs("add", "jetstack", "https://charts.jetstack.io", "--force-update")); err != nil {
			return ctx, fmt.Errorf("adding jetstack helm repo: %w", err)
		}
		if err := h.RunInstall(
			helm.WithName("cert-manager"),
			helm.WithChart("jetstack/cert-manager"),
			helm.WithNamespace("cert-manager"),
			helm.WithArgs("--create-namespace", "--set", "crds.enabled=true"),
			helm.WithWait(),
			helm.WithTimeout("5m"),
		); err != nil {
			return ctx, fmt.Errorf("installing cert-manager: %w", err)
		}
		if err := h.RunUpgrade(
			helm.WithName("trust-manager"),
			helm.WithChart("oci://quay.io/jetstack/charts/trust-manager"),
			helm.WithNamespace("cert-manager"),
			helm.WithArgs("--install"),
			helm.WithWait(),
			helm.WithTimeout("5m"),
		); err != nil {
			return ctx, fmt.Errorf("installing trust-manager: %w", err)
		}
		return ctx, nil
	}
}

func installOperator() env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		root, err := repoRoot()
		if err != nil {
			return ctx, err
		}

		h := helm.New(cfg.KubeconfigFile())
		if err := h.RunInstall(
			helm.WithName("netobserv"),
			helm.WithChart(filepath.Join(root, "helm")),
			helm.WithNamespace(Namespace),
			helm.WithArgs(
				"--create-namespace",
				"--set", "operator.image="+operatorImage,
				"--set", "operator.version="+operatorVersion,
			),
		); err != nil {
			return ctx, fmt.Errorf("installing netobserv chart: %w", err)
		}

		// The chart hardcodes imagePullPolicy=Always, which is right for
		// production (mutable tags) but would make the kubelet try, and
		// fail, to pull our locally-built, Kind-loaded image from a real
		// registry. Patch it to use the local image instead.
		patch := `{"spec":{"template":{"spec":{"containers":[{"name":"manager","imagePullPolicy":"IfNotPresent"}]}}}}`
		if err := run(root, nil, "kubectl", "--kubeconfig", cfg.KubeconfigFile(), "-n", Namespace,
			"patch", "deployment", "netobserv-controller-manager", "-p", patch); err != nil {
			return ctx, fmt.Errorf("patching operator image pull policy: %w", err)
		}
		return ctx, nil
	}
}
