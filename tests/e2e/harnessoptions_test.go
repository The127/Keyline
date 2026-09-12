//go:build e2e

package e2e

func withPort(port int) harnessOption {
	return func(options *harnessOptions) {
		options.port = port
	}
}
