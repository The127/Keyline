package jobs

import (
	"Keyline/internal/clock"
	"Keyline/internal/config"
	"Keyline/internal/services"
	"Keyline/internal/services/mocks"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type KeyRotateJobSuite struct {
	suite.Suite
}

func TestKeyRotateJobSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(KeyRotateJobSuite))
}

func (s *KeyRotateJobSuite) TestDeletesExpiredKeys() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	clockService, clockSetter := clock.NewMockServiceNow()

	clockSetter(time.Now().Add(-time.Hour * 24 * 365 * 10))
	signingAlgorithm := config.SigningAlgorithmEdDSA
	keyPair, err := services.GetKeyStrategy(signingAlgorithm).Generate(clockService)
	s.Require().NoError(err)

	clockSetter(time.Now())

	keys := []services.KeyPair{
		keyPair,
	}
	keyStore := mocks.NewMockKeyStore(ctrl)
	keyStore.EXPECT().Remove("vs-name", signingAlgorithm, keyPair.ComputeKid()).Return(nil)

	// act
	err = deleteExpiredKeys(keys, keyStore, "vs-name", clockService.Now())

	// assert
	s.Require().NoError(err)
}
