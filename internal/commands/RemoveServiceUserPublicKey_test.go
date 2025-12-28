package commands

import (
	"Keyline/internal/database"
	"Keyline/internal/middlewares"
	mocks2 "Keyline/internal/mocks"
	"Keyline/internal/repositories"
	"Keyline/internal/repositories/mocks"
	"Keyline/utils"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/The127/ioc"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type RemoveServiceUserPublicKeyCommandSuite struct {
	suite.Suite
}

func TestRemoveServiceUserPublicKeyCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RemoveServiceUserPublicKeyCommandSuite))
}

func (s *RemoveServiceUserPublicKeyCommandSuite) createContext(
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

func (s *RemoveServiceUserPublicKeyCommandSuite) TestHappyPath() {
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

	serviceUser := repositories.NewServiceUser("service-user", virtualServer.Id())
	serviceUser.Mock(now)
	userRepository := mocks.NewMockUserRepository(ctrl)
	userRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Cond(func(x *repositories.UserFilter) bool {
		return x.GetId() == serviceUser.Id() && x.GetVirtualServerId() == virtualServer.Id() && x.GetServiceUser() == true
	})).Return(serviceUser, nil)

	credential := repositories.NewCredential(serviceUser.Id(), &repositories.CredentialServiceUserKey{
		PublicKey: "publicKey",
	})
	credential.Mock(now)
	credentialRepository := mocks.NewMockCredentialRepository(ctrl)
	credentialRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Cond(func(x *repositories.CredentialFilter) bool {
		return x.GetDetailPublicKey() == "publicKey" &&
			x.GetUserId() == serviceUser.Id() &&
			x.GetType() == repositories.CredentialTypeServiceUserKey
	})).Return(credential, nil)
	credentialRepository.EXPECT().Delete(gomock.Any())

	ctx := s.createContext(ctrl, virtualServerRepository, userRepository, credentialRepository)
	cmd := RemoveServiceUserPublicKey{
		VirtualServerName: "virtualServer",
		ServiceUserId:     serviceUser.Id(),
		PublicKey:         "publicKey",
	}

	// act
	resp, err := HandleRemoveServiceUserPublicKey(ctx, cmd)

	// assert
	s.Require().NoError(err)
	s.NotNil(resp)
}

func (s *RemoveServiceUserPublicKeyCommandSuite) TestCredentialError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	serviceUser := repositories.NewServiceUser("service-user", virtualServer.Id())
	userRepository := mocks.NewMockUserRepository(ctrl)
	userRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(serviceUser, nil)

	credentialRepository := mocks.NewMockCredentialRepository(ctrl)
	credentialRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, virtualServerRepository, userRepository, credentialRepository)
	cmd := RemoveServiceUserPublicKey{}

	// act
	resp, err := HandleRemoveServiceUserPublicKey(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}

func (s *RemoveServiceUserPublicKeyCommandSuite) TestUserError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	userRepository := mocks.NewMockUserRepository(ctrl)
	userRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, virtualServerRepository, userRepository, nil)
	cmd := RemoveServiceUserPublicKey{}

	// act
	resp, err := HandleRemoveServiceUserPublicKey(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}

func (s *RemoveServiceUserPublicKeyCommandSuite) TestVirtualServerError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(nil, errors.New("error"))

	ctx := s.createContext(ctrl, virtualServerRepository, nil, nil)
	cmd := RemoveServiceUserPublicKey{}

	// act
	resp, err := HandleRemoveServiceUserPublicKey(ctx, cmd)

	// assert
	s.Require().Error(err)
	s.Nil(resp)
}
