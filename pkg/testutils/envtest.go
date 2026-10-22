package testutils

import (
	"os"
	"path/filepath"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func GetFirstFoundEnvTestBinaryDir() string {
	basePath := filepath.Join("..", "..", "..", "bin", "k8s")
	entries, err := os.ReadDir(basePath)

	if err != nil {
		logf.Log.Error(err, "Failed to read directory", "path", basePath)
		return ""
	}

	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(basePath, entry.Name())
		}
	}

	return ""
}

// StartEnvTest starts env and registers its Stop with DeferCleanup, so the
// control plane stops after AfterSuite even when AfterSuite fails.
// Call it inside a Ginkgo node.
func StartEnvTest(env *envtest.Environment) *rest.Config {
	ginkgo.GinkgoHelper()

	cfg, err := env.Start()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(cfg).NotTo(gomega.BeNil())

	ginkgo.DeferCleanup(env.Stop)

	return cfg
}
