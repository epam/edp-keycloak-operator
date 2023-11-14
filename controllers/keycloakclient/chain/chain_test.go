package chain

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestMake(t *testing.T) {
	chain := Make(runtime.NewScheme(), fake.NewClientBuilder().Build(), logr.Discard())
	require.NotNil(t, chain)
}
