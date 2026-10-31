package v1

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const upgradeRepoRoot = "../.."

var (
	upgradeHeadingRe = regexp.MustCompile(`(?m)^## From (\d+\.\d+)\.x to (\d+\.\d+)\.x$`)
	upgradeSectionRe = regexp.MustCompile(`(?m)^## `)
	upgradeFromRe    = regexp.MustCompile(`(?m)^## From .*$`)
	upgradeCRDLineRe = regexp.MustCompile(`(?m)^kubectl replace -f https://raw\.githubusercontent\.com/epam/edp-keycloak-operator/(\S+)/deploy-templates/crds/(\S+\.yaml)$`)
)

// TestUpgradeNotesPinCRDsOfNewestRelease requires the newest upgrade section that updates CRDs to list
// every file of deploy-templates/crds exactly once, pinned to the release tag.
func TestUpgradeNotesPinCRDsOfNewestRelease(t *testing.T) {
	page, err := os.ReadFile(upgradeRepoRoot + "/docs/upgrade-notes.md")
	require.NoError(t, err)

	content := string(page)

	for _, line := range upgradeFromRe.FindAllString(content, -1) {
		if strings.HasPrefix(line, "## From Keycloak ") {
			continue
		}

		require.Regexp(t, upgradeHeadingRe, line, "heading %q must match '## From <a>.x to <b>.x'", line)
	}

	headings := upgradeHeadingRe.FindAllStringSubmatchIndex(content, -1)
	require.NotEmpty(t, headings, "docs/upgrade-notes.md has no '## From <a>.x to <b>.x' section")

	var (
		target  string
		refs    [][]string
		section string
	)

	for _, h := range headings {
		body := content[h[1]:]
		if next := upgradeSectionRe.FindStringIndex(body); next != nil {
			body = body[:next[0]]
		}

		if m := upgradeCRDLineRe.FindAllStringSubmatch(body, -1); len(m) > 0 {
			target, refs, section = content[h[4]:h[5]], m, content[h[0]:h[1]]

			break
		}
	}

	require.NotEmpty(t, refs, "no section has 'kubectl replace -f' lines for deploy-templates/crds")

	tag := regexp.MustCompile(`^v` + regexp.QuoteMeta(target) + `\.\d+$`)

	got := make([]string, 0, len(refs))

	for _, r := range refs {
		require.Regexp(t, tag, r[1], "%s: CRD URL must be pinned to a v%s.<patch> tag", section, target)

		got = append(got, r[2])
	}

	entries, err := os.ReadDir(upgradeRepoRoot + "/deploy-templates/crds")
	require.NoError(t, err)

	want := make([]string, 0, len(entries))

	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
			want = append(want, e.Name())
		}
	}

	require.ElementsMatch(t, want, got, "%s must list every file of deploy-templates/crds exactly once", section)
}
