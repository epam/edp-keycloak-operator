package v1

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	repoRoot         = "../../.."
	webhookConfigDir = repoRoot + "/config/webhook"
)

var namespacePatches = map[string]string{
	"MutatingWebhookConfiguration":   "mutating_namespace_patch.yaml",
	"ValidatingWebhookConfiguration": "validating_namespace_patch.yaml",
}

// decodeYAML calls fn for every document in path.
func decodeYAML(t *testing.T, path string, fn func(decode func(any) error) error) {
	t.Helper()

	f, err := os.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, f.Close()) })

	decoder := yaml.NewYAMLOrJSONDecoder(f, 4096)

	for {
		if err := fn(decoder.Decode); err != nil {
			if errors.Is(err, io.EOF) {
				return
			}

			require.NoError(t, err)
		}
	}
}

// webhooks returns the webhooks of every document of kind in path.
func webhooks(t *testing.T, path, kind string) []admissionregistrationv1.ValidatingWebhook {
	t.Helper()

	var all []admissionregistrationv1.ValidatingWebhook

	decodeYAML(t, path, func(decode func(any) error) error {
		var doc struct {
			Kind     string                                      `json:"kind"`
			Webhooks []admissionregistrationv1.ValidatingWebhook `json:"webhooks"`
		}

		if err := decode(&doc); err != nil {
			return err
		}

		if doc.Kind == kind {
			all = append(all, doc.Webhooks...)
		}

		return nil
	})

	return all
}

func sortedNames(hooks []admissionregistrationv1.ValidatingWebhook) []string {
	names := make([]string, 0, len(hooks))
	for _, w := range hooks {
		names = append(names, w.Name)
	}

	slices.Sort(names)

	return names
}

// generatedWebhookNames returns every webhook name controller-gen writes to manifests.yaml.
func generatedWebhookNames(t *testing.T) []string {
	t.Helper()

	names := make([]string, 0, len(namespacePatches))
	for kind := range namespacePatches {
		names = append(names, sortedNames(webhooks(t, filepath.Join(webhookConfigDir, "manifests.yaml"), kind))...)
	}

	slices.Sort(names)

	return names
}

// TestWebhookNamespacePatchesCoverEveryWebhook requires each namespace patch to list exactly the
// webhooks of its kind in manifests.yaml, each with a namespaceSelector, and kustomization.yaml to apply both.
func TestWebhookNamespacePatchesCoverEveryWebhook(t *testing.T) {
	generated := filepath.Join(webhookConfigDir, "manifests.yaml")

	for kind, patch := range namespacePatches {
		t.Run(kind, func(t *testing.T) {
			want := sortedNames(webhooks(t, generated, kind))
			require.NotEmpty(t, want)

			patched := webhooks(t, filepath.Join(webhookConfigDir, patch), kind)
			require.Equal(t, want, sortedNames(patched), "%s must list every %s webhook in %s", patch, kind, generated)

			for _, w := range patched {
				require.NotNil(t, w.NamespaceSelector, "%s: %s has no namespaceSelector", patch, w.Name)
				require.NotEmpty(t, w.NamespaceSelector.MatchExpressions, "%s: %s has an empty namespaceSelector", patch, w.Name)
			}
		})
	}

	t.Run("kustomization applies both patches", func(t *testing.T) {
		var kustomization struct {
			Patches []struct {
				Path string `json:"path"`
			} `json:"patches"`
		}

		decodeYAML(t, filepath.Join(webhookConfigDir, "kustomization.yaml"), func(decode func(any) error) error {
			return decode(&kustomization)
		})

		paths := make([]string, 0, len(kustomization.Patches))
		for _, p := range kustomization.Patches {
			paths = append(paths, p.Path)
		}

		for _, patch := range namespacePatches {
			require.Contains(t, paths, patch)
		}
	})
}

// TestHandWrittenWebhookListsMatchGenerated requires the Helm chart and the bundle CSV to declare
// exactly the webhooks in manifests.yaml.
func TestHandWrittenWebhookListsMatchGenerated(t *testing.T) {
	want := generatedWebhookNames(t)

	for _, tc := range []struct {
		path    string
		pattern *regexp.Regexp
	}{
		{repoRoot + "/deploy-templates/templates/webhook/manifest.yaml", regexp.MustCompile(`(?m)^\s+name: (\S+\.kb\.io)\s*$`)},
		{repoRoot + "/bundle/manifests/edp-keycloak-operator.clusterserviceversion.yaml", regexp.MustCompile(`(?m)^\s+generateName: (\S+\.kb\.io)\s*$`)},
	} {
		t.Run(filepath.Base(tc.path), func(t *testing.T) {
			content, err := os.ReadFile(tc.path)
			require.NoError(t, err)

			matches := tc.pattern.FindAllStringSubmatch(string(content), -1)
			got := make([]string, 0, len(matches))

			for _, m := range matches {
				got = append(got, m[1])
			}

			slices.Sort(got)
			require.Equal(t, want, got, "%s must declare every webhook in config/webhook/manifests.yaml", tc.path)
		})
	}
}
