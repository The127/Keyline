package commands

import (
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type RemoveApplicationKeyCommandSuite struct {
	suite.Suite
}

func TestRemoveApplicationKeyCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RemoveApplicationKeyCommandSuite))
}

func (s *RemoveApplicationKeyCommandSuite) TestHappyPath() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	f := arrangeApplicationKeyFixture(s.T(), ctrl, repositories.TokenEndpointAuthMethodPrivateKeyJwt)
	key := repositories.NewApplicationKey(f.application.Id(), "key-1", testPublicKeyPem())
	f.keys.EXPECT().FirstOrErr(gomock.Any(), gomock.Cond(func(x *repositories.ApplicationKeyFilter) bool {
		return x.GetApplicationId() == f.application.Id() && x.GetKid() == "key-1"
	})).Return(key, nil)
	f.keys.EXPECT().Delete(key.Id())

	// act
	_, err := HandleRemoveApplicationKey(f.ctx, RemoveApplicationKey{
		VirtualServerName: "virtualServer",
		ProjectSlug:       "project",
		ApplicationId:     f.application.Id(),
		Kid:               "key-1",
	})

	// assert
	s.Require().NoError(err)
}

func (s *RemoveApplicationKeyCommandSuite) TestUnknownKid() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	f := arrangeApplicationKeyFixture(s.T(), ctrl, repositories.TokenEndpointAuthMethodPrivateKeyJwt)
	f.keys.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(nil, utils.ErrApplicationKeyNotFound)

	// act
	_, err := HandleRemoveApplicationKey(f.ctx, RemoveApplicationKey{
		VirtualServerName: "virtualServer",
		ProjectSlug:       "project",
		ApplicationId:     f.application.Id(),
		Kid:               "missing",
	})

	// assert
	s.Require().ErrorIs(err, utils.ErrHttpNotFound)
}
