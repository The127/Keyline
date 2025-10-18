package services

import (
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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			// arrange
			strategy := GetKeyStrategy(test.alg)

			// act
			privateKey, publicKey, err := strategy.Generate()
			require.NoError(t, err)
			require.NotEmpty(t, privateKey)
			require.NotEmpty(t, publicKey)

			exported, err := strategy.Export(privateKey)
			require.NoError(t, err)

			importedPriv, importedPub, err := strategy.Import(exported)
			require.NoError(t, err)
			require.Equal(t, privateKey, importedPriv)
			require.Equal(t, publicKey, importedPub)
		})
	}
}
