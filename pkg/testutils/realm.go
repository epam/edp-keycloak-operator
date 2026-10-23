package testutils

import (
	"fmt"
	"os"
	"strconv"
	"sync/atomic"
	"time"
)

// realmSuffix is fixed for the lifetime of one test binary and differs between
// concurrent test binaries. Base 36 keeps generated names short.
var realmSuffix = fmt.Sprintf(
	"%s-%s",
	strconv.FormatInt(int64(os.Getpid()), 36),
	strconv.FormatInt(time.Now().UnixNano(), 36),
)

var uniqueNameCounter atomic.Uint64

// RealmName appends the per-process suffix to base.
// Every integration suite shares one Keycloak at TEST_KEYCLOAK_URL and realm
// names are global there. One name per base per test binary: use it for names
// that several specs, or a CR and its realm, share.
// The result is a valid Kubernetes object name, so it also serves as a CR name.
func RealmName(base string) string {
	return fmt.Sprintf("%s-%s", base, realmSuffix)
}

// UniqueName appends the per-process suffix and a per-call counter to prefix.
// Every call returns a new name, also from parallel tests: use it for objects
// that a single test creates.
// Prefix: lowercase letters, digits and hyphens. The result is a DNS-1123 label
// when the prefix is at most 32 characters.
func UniqueName(prefix string) string {
	return RealmName(prefix) + "-" + strconv.FormatUint(uniqueNameCounter.Add(1), 36)
}
