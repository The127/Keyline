//go:build integration
// +build integration

package integration

import (
	"Keyline/internal/logging"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	logging.Init()
	RunSpecs(t, "Integration Suite")
}
