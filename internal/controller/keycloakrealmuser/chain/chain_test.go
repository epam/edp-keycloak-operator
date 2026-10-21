package chain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeChain_VerifyUserAttributesOrder(t *testing.T) {
	ch := MakeChain(nil, nil)

	indexOf := func(target RealmUserHandler) int {
		for i, h := range ch.handlers {
			switch target.(type) {
			case *SyncUserIdentityProviders:
				if _, ok := h.(*SyncUserIdentityProviders); ok {
					return i
				}
			case *VerifyUserAttributes:
				if _, ok := h.(*VerifyUserAttributes); ok {
					return i
				}
			case *CleanupResource:
				if _, ok := h.(*CleanupResource); ok {
					return i
				}
			}
		}

		return -1
	}

	syncIdPs := indexOf(&SyncUserIdentityProviders{})
	verify := indexOf(&VerifyUserAttributes{})
	cleanup := indexOf(&CleanupResource{})

	require.NotEqual(t, -1, syncIdPs, "SyncUserIdentityProviders missing from chain")
	require.NotEqual(t, -1, verify, "VerifyUserAttributes missing from chain")
	require.NotEqual(t, -1, cleanup, "CleanupResource missing from chain")

	assert.Less(t, syncIdPs, verify, "VerifyUserAttributes must follow SyncUserIdentityProviders")
	assert.Less(t, verify, cleanup, "VerifyUserAttributes must precede CleanupResource")
}
