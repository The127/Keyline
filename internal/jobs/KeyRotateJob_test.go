package jobs

import (
	"github.com/The127/Keyline/config"
	"github.com/The127/Keyline/internal/logging"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/internal/services"
	"github.com/The127/Keyline/internal/services/mocks"
	"testing"
	"time"

	"github.com/The127/go-clock"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type KeyRotateJobSuite struct {
	suite.Suite
}

func TestKeyRotateJobSuite(t *testing.T) {
	t.Parallel()
	logging.Init()
	suite.Run(t, new(KeyRotateJobSuite))
}

func (s *KeyRotateJobSuite) TestDeletesExpiredKeys() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	clockService, clockSetter := clock.NewMockClock(time.Now())

	clockSetter(time.Now().Add(-time.Hour * 24 * 365 * 10))
	signingAlgorithm := config.SigningAlgorithmEdDSA
	keyPair, err := services.GetKeyStrategy(signingAlgorithm).Generate(clockService)
	s.Require().NoError(err)

	clockSetter(time.Now())

	keys := []services.KeyPair{
		keyPair,
	}
	keyStore := mocks.NewMockKeyStore(ctrl)
	keyStore.EXPECT().Remove("vs-name", signingAlgorithm, keyPair.GetKid()).Return(nil)

	// act
	err = deleteExpiredKeys(keys, keyStore, "vs-name", clockService.Now())

	// assert
	s.Require().NoError(err)
}

func (s *KeyRotateJobSuite) TestDeletesOrphanKeys() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	clockService, _ := clock.NewMockClock(time.Now())

	eddsaKey, err := services.GetKeyStrategy(config.SigningAlgorithmEdDSA).Generate(clockService)
	s.Require().NoError(err)
	rs256Key, err := services.GetKeyStrategy(config.SigningAlgorithmRS256).Generate(clockService)
	s.Require().NoError(err)

	// VS is configured with EdDSA only; RS256 key is an orphan
	vs := repositories.NewVirtualServer("vs-name", "VS Name")
	vs.SetPrimarySigningAlgorithm(config.SigningAlgorithmEdDSA)
	vs.SetAdditionalSigningAlgorithms([]config.SigningAlgorithm{})

	keys := []services.KeyPair{eddsaKey, rs256Key}
	keyStore := mocks.NewMockKeyStore(ctrl)
	keyStore.EXPECT().Remove("vs-name", config.SigningAlgorithmRS256, rs256Key.GetKid()).Return(nil)

	// act
	err = deleteOrphanKeys(keys, keyStore, vs)

	// assert
	s.Require().NoError(err)
}

func (s *KeyRotateJobSuite) TestDoesNotDeleteConfiguredKeys() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	clockService, _ := clock.NewMockClock(time.Now())

	eddsaKey, err := services.GetKeyStrategy(config.SigningAlgorithmEdDSA).Generate(clockService)
	s.Require().NoError(err)
	rs256Key, err := services.GetKeyStrategy(config.SigningAlgorithmRS256).Generate(clockService)
	s.Require().NoError(err)

	// VS is configured with both algorithms — nothing is an orphan
	vs := repositories.NewVirtualServer("vs-name", "VS Name")
	vs.SetPrimarySigningAlgorithm(config.SigningAlgorithmEdDSA)
	vs.SetAdditionalSigningAlgorithms([]config.SigningAlgorithm{config.SigningAlgorithmRS256})

	keys := []services.KeyPair{eddsaKey, rs256Key}
	keyStore := mocks.NewMockKeyStore(ctrl) // no Remove expected

	// act
	err = deleteOrphanKeys(keys, keyStore, vs)

	// assert
	s.Require().NoError(err)
}
