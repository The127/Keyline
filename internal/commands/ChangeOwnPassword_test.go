package commands

import (
	"Keyline/internal/database"
	"Keyline/internal/middlewares"
	"Keyline/internal/mocks"
	"Keyline/internal/repositories"
	repoMocks "Keyline/internal/repositories/mocks"
	"Keyline/utils"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/The127/ioc"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type ChangeOwnPasswordCommandSuite struct {
	suite.Suite
}

func TestChangeOwnPasswordCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ChangeOwnPasswordCommandSuite))
}

func (s *ChangeOwnPasswordCommandSuite) createContext(ctrl *gomock.Controller, cr repositories.CredentialRepository) context.Context {
	dc := ioc.NewDependencyCollection()

	dbContext := mocks.NewMockContext(ctrl)
	ioc.RegisterScoped(dc, func(_ *ioc.DependencyProvider) database.Context {
		return dbContext
	})

	if cr != nil {
		dbContext.EXPECT().Credentials().Return(cr).AnyTimes()
	}

	scope := dc.BuildProvider()
	s.T().Cleanup(func() {
		utils.PanicOnError(scope.Close, "closing scope")
	})

	return middlewares.ContextWithScope(s.T().Context(), scope)
}

func (s *ChangeOwnPasswordCommandSuite) TestCredentialNotFound() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	cr := repoMocks.NewMockCredentialRepository(ctrl)
	cr.EXPECT().FirstOrNil(gomock.Any(), gomock.Any()).Return(nil, nil)

	ctx := s.createContext(ctrl, cr)
	_, err := HandleChangeOwnPassword(ctx, ChangeOwnPassword{
		CurrentPassword: "old",
		NewPassword:     "new",
	})
	s.Error(err)
}

func (s *ChangeOwnPasswordCommandSuite) TestCredentialLookupError() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	cr := repoMocks.NewMockCredentialRepository(ctrl)
	cr.EXPECT().FirstOrNil(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))

	ctx := s.createContext(ctrl, cr)
	_, err := HandleChangeOwnPassword(ctx, ChangeOwnPassword{
		CurrentPassword: "old",
		NewPassword:     "new",
	})
	s.Error(err)
}

func (s *ChangeOwnPasswordCommandSuite) TestWrongCurrentPassword() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()
	now := time.Now()

	hashedPassword := utils.HashPassword("correct-password")
	cred := repositories.NewCredential(
		repositories.NewBaseModel().Id(),
		&repositories.CredentialPasswordDetails{
			HashedPassword: hashedPassword,
			Temporary:      false,
		},
	)
	cred.Mock(now)

	cr := repoMocks.NewMockCredentialRepository(ctrl)
	cr.EXPECT().FirstOrNil(gomock.Any(), gomock.Any()).Return(cred, nil)

	ctx := s.createContext(ctrl, cr)
	_, err := HandleChangeOwnPassword(ctx, ChangeOwnPassword{
		CurrentPassword: "wrong-password",
		NewPassword:     "new-password",
	})
	s.Error(err)
}

func (s *ChangeOwnPasswordCommandSuite) TestHappyPath() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()
	now := time.Now()

	hashedPassword := utils.HashPassword("current-password")
	cred := repositories.NewCredential(
		repositories.NewBaseModel().Id(),
		&repositories.CredentialPasswordDetails{
			HashedPassword: hashedPassword,
			Temporary:      false,
		},
	)
	cred.Mock(now)

	cr := repoMocks.NewMockCredentialRepository(ctrl)
	cr.EXPECT().FirstOrNil(gomock.Any(), gomock.Any()).Return(cred, nil)
	cr.EXPECT().Update(gomock.Any())

	ctx := s.createContext(ctrl, cr)
	resp, err := HandleChangeOwnPassword(ctx, ChangeOwnPassword{
		CurrentPassword: "current-password",
		NewPassword:     "new-password",
	})
	s.Require().NoError(err)
	s.NotNil(resp)
}
