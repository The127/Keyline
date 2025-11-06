//go:build e2e

package e2e

import (
	"Keyline/internal/logging"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	logging.Init()
	RunSpecs(t, "e2e Suite")
}
