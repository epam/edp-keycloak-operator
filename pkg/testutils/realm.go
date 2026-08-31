package testutils

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// realmSuffix is fixed for the lifetime of one test binary and differs between
// concurrent test binaries. Base 36 keeps generated names short.
var realmSuffix = fmt.Sprintf(
	"%s-%s",
	strconv.FormatInt(int64(os.Getpid()), 36),
	strconv.FormatInt(time.Now().UnixNano(), 36),
)

// RealmName appends the per-process suffix to base.
// Every integration suite shares one Keycloak at TEST_KEYCLOAK_URL and realm
// names are global there. All test realm names must go through this function.
// The result is a valid Kubernetes object name, so it also serves as a CR name.
func RealmName(base string) string {
	return fmt.Sprintf("%s-%s", base, realmSuffix)
}
