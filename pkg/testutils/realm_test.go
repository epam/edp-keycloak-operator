package testutils

import (
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUniqueName_ConcurrentCallsAreDistinct(t *testing.T) {
	t.Parallel()

	const (
		goroutines = 64
		calls      = 1000
	)

	results := make([][]string, goroutines)

	var wg sync.WaitGroup

	for i := range goroutines {
		wg.Add(1)

		go func() {
			defer wg.Done()

			names := make([]string, 0, calls)
			for range calls {
				names = append(names, UniqueName("p"))
			}

			results[i] = names
		}()
	}

	wg.Wait()

	seen := make(map[string]struct{}, goroutines*calls)

	for _, names := range results {
		for _, name := range names {
			seen[name] = struct{}{}
		}
	}

	require.Len(t, seen, goroutines*calls)
}

func TestUniqueName_Format(t *testing.T) {
	t.Parallel()

	dns1123Label := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

	for _, prefix := range []string{
		"p",
		"test-realm-org",
		"test-realm-cp-policies",
		"test-realm-create-user-conflict",
		strings.Repeat("a", 32),
	} {
		t.Run(prefix, func(t *testing.T) {
			t.Parallel()

			got := UniqueName(prefix)

			require.Regexp(t, "^"+regexp.QuoteMeta(prefix)+"-[0-9a-z]+-[0-9a-z]+-[0-9a-z]+$", got)
			require.Regexp(t, dns1123Label, got)
			require.LessOrEqual(t, len(got), 63)
		})
	}
}

func TestUniqueName_SharesProcessSuffixWithRealmName(t *testing.T) {
	t.Parallel()

	require.True(t, strings.HasPrefix(UniqueName("x"), RealmName("x")+"-"))
}

func TestUniqueName_CounterIncreasesWithinGoroutine(t *testing.T) {
	t.Parallel()

	const calls = 1000

	var previous uint64

	for i := range calls {
		got := UniqueName("p")

		counter, err := strconv.ParseUint(got[strings.LastIndex(got, "-")+1:], 36, 64)
		require.NoError(t, err)

		if i > 0 {
			require.Greater(t, counter, previous)
		}

		previous = counter
	}
}
