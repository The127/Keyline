package commands

import (
	"Keyline/internal/database"
	"Keyline/internal/middlewares"
	mocks2 "Keyline/internal/mocks"
	"Keyline/internal/repositories"
	"Keyline/internal/repositories/mocks"
	"Keyline/utils"
	"context"
	"testing"
	"time"

	"github.com/The127/ioc"

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
	ctrl *gomock.Controller,
	virtualServerRepository repositories.VirtualServerRepository,
	userRepository repositories.UserRepository,
	credentialRepository repositories.CredentialRepository,
) context.Context {
	dc := ioc.NewDependencyCollection()

	dbContext := mocks2.NewMockContext(ctrl)
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) database.Context {
		return dbContext
	})

	if virtualServerRepository != nil {
		dbContext.EXPECT().VirtualServers().Return(virtualServerRepository).AnyTimes()
	}

	if userRepository != nil {
		dbContext.EXPECT().Users().Return(userRepository).AnyTimes()
	}

	if credentialRepository != nil {
		dbContext.EXPECT().Credentials().Return(credentialRepository).AnyTimes()
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
	virtualServerRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Cond(func(x *repositories.VirtualServerFilter) bool {
		return x.GetName() == "virtualServer"
	})).Return(virtualServer, nil)

	user := repositories.NewUser("user", "User", "user@mail", virtualServer.Id())
	user.Mock(now)
	userRepository := mocks.NewMockUserRepository(ctrl)
	userRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Cond(func(x *repositories.UserFilter) bool {
		return x.GetId() == user.Id()
	})).Return(user, nil)

	credentialRepository := mocks.NewMockCredentialRepository(ctrl)
	credentialRepository.EXPECT().Insert(gomock.Cond(func(x *repositories.Credential) bool {
		return x.UserId() == user.Id() &&
			x.Type() == repositories.CredentialTypeServiceUserKey &&
			utils.Unwrap(x.ServiceUserKeyDetails()).PublicKey == "publicKey"
	}))

	ctx := s.createContext(ctrl, virtualServerRepository, userRepository, credentialRepository)
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
