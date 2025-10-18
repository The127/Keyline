package services

import (
	"Keyline/internal/clock"
	"Keyline/internal/config"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKeyStrategy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		alg  config.SigningAlgorithm
	}{
		{"RS256", config.SigningAlgorithmRS256},
		{"EdDSA", config.SigningAlgorithmEdDSA},
	}

	clockService, _ := clock.NewMockServiceNow()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			// arrange
			strategy := GetKeyStrategy(test.alg)

			// act
			keyPair, err := strategy.Generate(clockService)
			require.NoError(t, err)

			exported, err := strategy.Export(keyPair.PrivateKey())
			require.NoError(t, err)

			importedPriv, importedPub, err := strategy.Import(exported)
			require.NoError(t, err)
			require.Equal(t, keyPair.PrivateKey(), importedPriv)
			require.Equal(t, keyPair.PublicKey(), importedPub)
		})
	}
}
