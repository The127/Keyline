package commands

import (
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/internal/repositories/mocks"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type AssociateServiceUserPublicKeyCommandSuite struct {
	suite.Suite
}

func TestAssociateServiceUserPublicKeyCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AssociateServiceUserPublicKeyCommandSuite))
}

func (s *AssociateServiceUserPublicKeyCommandSuite) createContext(
	virtualServerRepository repositories.VirtualServerRepository,
	userRepository repositories.UserRepository,
	credentialRepository repositories.CredentialRepository,
) context.Context {
	dc := ioc.NewDependencyCollection()

	if virtualServerRepository != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.VirtualServerRepository {
			return virtualServerRepository
		})
	}

	if userRepository != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.UserRepository {
			return userRepository
		})
	}

	if credentialRepository != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.CredentialRepository {
			return credentialRepository
		})
	}

	scope := dc.BuildProvider()
	s.T().Cleanup(func() {
		utils.PanicOnError(scope.Close, "closing scope")
	})

	return middlewares.ContextWithScope(s.T().Context(), scope)
}

func (s *AssociateServiceUserPublicKeyCommandSuite) TestHappyPath() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories.VirtualServerFilter) bool {
		return x.GetName() == "virtualServer"
	})).Return(virtualServer, nil)

	user := repositories.NewUser("user", "User", "user@mail", virtualServer.Id())
	user.Mock(now)
	userRepository := mocks.NewMockUserRepository(ctrl)
	userRepository.EXPECT().Single(gomock.Any(), gomock.Cond(func(x repositories.UserFilter) bool {
		return x.GetId() == user.Id()
	})).Return(user, nil)

	credentialRepository := mocks.NewMockCredentialRepository(ctrl)
	credentialRepository.EXPECT().Insert(gomock.Any(), gomock.Cond(func(x *repositories.Credential) bool {
		return x.UserId() == user.Id() &&
			x.Type() == repositories.CredentialTypeServiceUserKey &&
			utils.Unwrap(x.ServiceUserKeyDetails()).PublicKey == "publicKey"
	})).Return(nil)

	ctx := s.createContext(virtualServerRepository, userRepository, credentialRepository)
	cmd := AssociateServiceUserPublicKey{
		VirtualServerName: "virtualServer",
		ServiceUserId:     user.Id(),
		PublicKey:         "publicKey",
	}

	// act
	resp, err := HandleAssociateServiceUserPublicKey(ctx, cmd)

	// assert
	s.Require().NoError(err)
	s.NotNil(resp)
}
