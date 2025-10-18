package commands

import (
	"Keyline/internal/middlewares"
	"Keyline/internal/repositories"
	"Keyline/internal/repositories/mocks"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type DeleteApplicationCommandSuite struct {
	suite.Suite
}

func TestDeleteApplicationCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(DeleteApplicationCommandSuite))
}

func (s *DeleteApplicationCommandSuite) createContext(
	virtualServerRepository repositories.VirtualServerRepository,
	applicationRepository repositories.ApplicationRepository,
) context.Context {
	dc := ioc.NewDependencyCollection()

	if virtualServerRepository != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.VirtualServerRepository {
			return virtualServerRepository
		})
	}

	if applicationRepository != nil {
		ioc.RegisterTransient(dc, func(_ *ioc.DependencyProvider) repositories.ApplicationRepository {
			return applicationRepository
		})
	}

	scope := dc.BuildProvider()
	s.T().Cleanup(func() {
		utils.PanicOnError(scope.Close, "closing scope")
	})

	return middlewares.ContextWithScope(s.T().Context(), scope)
}

func (s *DeleteApplicationCommandSuite) TestTryingToDeleteSystemApplication() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	application := repositories.NewApplication(virtualServer.Id(), "application", "Application", repositories.ApplicationTypePublic, []string{})
	application.Mock(now)
	application.SetSystemApplication(true)
	applicationRepository := mocks.NewMockApplicationRepository(ctrl)
	applicationRepository.EXPECT().First(gomock.Any(), gomock.Any()).Return(application, nil)

	ctx := s.createContext(virtualServerRepository, applicationRepository)
	cmd := DeleteApplication{}

	// act
	_, err := HandleDeleteApplication(ctx, cmd)

	// assert
	s.Error(err)
}

func (s *DeleteApplicationCommandSuite) TesttApplicationError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	now := time.Now()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	applicationRepository := mocks.NewMockApplicationRepository(ctrl)
	applicationRepository.EXPECT().First(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("error"))

	ctx := s.createContext(virtualServerRepository, applicationRepository)
	cmd := DeleteApplication{}

	// act
	_, err := HandleDeleteApplication(ctx, cmd)

	// assert
	s.Error(err)
}

func (s *DeleteApplicationCommandSuite) TestVirtualServerError() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().Single(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("error"))

	ctx := s.createContext(virtualServerRepository, nil)
	cmd := DeleteApplication{}

	// act
	_, err := HandleDeleteApplication(ctx, cmd)

	// assert
	s.Error(err)
}

func (s *DeleteApplicationCommandSuite) TestHappyPath() {
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

	application := repositories.NewApplication(virtualServer.Id(), "application", "Application", repositories.ApplicationTypePublic, []string{})
	application.Mock(now)
	applicationRepository := mocks.NewMockApplicationRepository(ctrl)
	applicationRepository.EXPECT().First(gomock.Any(), gomock.Cond(func(x repositories.ApplicationFilter) bool {
		return x.GetId() == application.Id()
	})).Return(application, nil)
	applicationRepository.EXPECT().Delete(gomock.Any(), application.Id()).Return(nil)

	ctx := s.createContext(virtualServerRepository, applicationRepository)
	cmd := DeleteApplication{
		VirtualServerName: "virtualServer",
		ApplicationId:     application.Id(),
	}

	// act
	response, err := HandleDeleteApplication(ctx, cmd)

	// assert
	s.Require().NoError(err)
	s.NotNil(response)
}
