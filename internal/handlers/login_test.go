package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/mocks"
	"github.com/The127/Keyline/internal/repositories"
	repoMocks "github.com/The127/Keyline/internal/repositories/mocks"
	"github.com/The127/Keyline/utils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestMaxFailedPasswordAttempts_LowEnoughForOnlineGuessingMitigation(t *testing.T) {
	t.Parallel()
	// Sanity-check the threshold against RFC 6819 §5.1.4.2.3 guidance: a
	// single anonymously-minted loginToken must not authorize a meaningful
	// brute-force budget. Keep this assertion strict so a future bump is a
	// deliberate, code-reviewed change.
	assert.LessOrEqual(t, MaxFailedPasswordAttempts, 10)
	assert.GreaterOrEqual(t, MaxFailedPasswordAttempts, 3)
}

func newPasswordVerifyTestContext(t *testing.T) (context.Context, *mocks.MockContext, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	dbContext := mocks.NewMockContext(ctrl)
	return context.Background(), dbContext, ctrl
}

func seedUserAndPasswordCredential(
	t *testing.T,
	dbContext *mocks.MockContext,
	ctrl *gomock.Controller,
	username string,
	password string,
) (*repositories.User, *repositories.Credential) {
	t.Helper()
	user := repositories.NewUser(username, "Display "+username, username+"@test", uuid.New())
	user.Mock(time.Now())

	userRepository := repoMocks.NewMockUserRepository(ctrl)
	userRepository.EXPECT().FirstOrNil(gomock.Any(), gomock.Any()).Return(user, nil).AnyTimes()
	dbContext.EXPECT().Users().Return(userRepository).AnyTimes()

	credential := repositories.NewCredential(user.Id(), &repositories.CredentialPasswordDetails{
		HashedPassword: utils.HashPassword(password),
		Temporary:      false,
	})
	credentialRepository := repoMocks.NewMockCredentialRepository(ctrl)
	credentialRepository.EXPECT().FirstOrNil(gomock.Any(), gomock.Any()).Return(credential, nil).AnyTimes()
	dbContext.EXPECT().Credentials().Return(credentialRepository).AnyTimes()

	return user, credential
}

func TestVerifyPasswordCredential_AcceptsCorrectPassword(t *testing.T) {
	t.Parallel()
	ctx, dbContext, ctrl := newPasswordVerifyTestContext(t)
	defer ctrl.Finish()
	expectedUser, _ := seedUserAndPasswordCredential(t, dbContext, ctrl, "alice", "correct-horse-battery-staple")

	user, ok, err := verifyPasswordCredential(ctx, dbContext, uuid.New(), "alice", "correct-horse-battery-staple")
	require.NoError(t, err)
	require.True(t, ok)
	require.NotNil(t, user)
	assert.Equal(t, expectedUser.Id(), user.Id())
}

func TestVerifyPasswordCredential_RejectsWrongPassword(t *testing.T) {
	t.Parallel()
	ctx, dbContext, ctrl := newPasswordVerifyTestContext(t)
	defer ctrl.Finish()
	seedUserAndPasswordCredential(t, dbContext, ctrl, "alice", "correct-horse-battery-staple")

	user, ok, err := verifyPasswordCredential(ctx, dbContext, uuid.New(), "alice", "wrong-password")
	require.NoError(t, err)
	require.False(t, ok)
	// User is still returned on a wrong-password to allow callers to log
	// the attempt against the user; the bool result is what gates auth.
	assert.NotNil(t, user)
}

func TestVerifyPasswordCredential_RejectsUnknownUser(t *testing.T) {
	t.Parallel()
	ctx, dbContext, ctrl := newUnknownUserContext(t)
	defer ctrl.Finish()

	user, ok, err := verifyPasswordCredential(ctx, dbContext, uuid.New(), "ghost", "anything")
	require.NoError(t, err)
	require.False(t, ok)
	assert.Nil(t, user)
}

func newUnknownUserContext(t *testing.T) (context.Context, *mocks.MockContext, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	dbContext := mocks.NewMockContext(ctrl)
	userRepository := repoMocks.NewMockUserRepository(ctrl)
	userRepository.EXPECT().FirstOrNil(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	dbContext.EXPECT().Users().Return(userRepository).AnyTimes()
	return context.Background(), dbContext, ctrl
}

// Compile-time check: verifyPasswordCredential's first parameter must be a
// database.Context, not the concrete mock; if this assignment fails to
// compile a refactor changed the helper's signature in a way the
// VerifyPassword caller can no longer satisfy.
var _ func(context.Context, database.Context, uuid.UUID, string, string) (*repositories.User, bool, error) = verifyPasswordCredential
