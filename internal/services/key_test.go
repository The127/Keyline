package services

import (
	"Keyline/internal/config"
	"testing"
	"time"

	"github.com/The127/go-clock"

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

	clockService, _ := clock.NewMockClock(time.Now())

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
