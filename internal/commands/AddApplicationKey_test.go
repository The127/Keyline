package commands

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"github.com/The127/Keyline/internal/database"
	"github.com/The127/Keyline/internal/middlewares"
	mocks2 "github.com/The127/Keyline/internal/mocks"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/internal/repositories/mocks"
	"github.com/The127/Keyline/utils"
	"testing"
	"time"

	"github.com/The127/ioc"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type AddApplicationKeyCommandSuite struct {
	suite.Suite
}

func TestAddApplicationKeyCommandSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(AddApplicationKeyCommandSuite))
}

func testPublicKeyPem() string {
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	return encodePublicKeyPem(publicKey)
}

func testRsaPublicKeyPem(bits int) string {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		panic(err)
	}
	return encodePublicKeyPem(&privateKey.PublicKey)
}

func encodePublicKeyPem(publicKey any) string {
	der, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		panic(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
}

type applicationKeyFixture struct {
	ctx         context.Context
	application *repositories.Application
	keys        *mocks.MockApplicationKeyRepository
}

func arrangeApplicationKeyFixture(t *testing.T, ctrl *gomock.Controller, method repositories.TokenEndpointAuthMethod) applicationKeyFixture {
	now := time.Now()

	virtualServer := repositories.NewVirtualServer("virtualServer", "Virtual Server")
	virtualServer.Mock(now)
	virtualServerRepository := mocks.NewMockVirtualServerRepository(ctrl)
	virtualServerRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(virtualServer, nil)

	project := repositories.NewProject(virtualServer.Id(), "project", "Project", "Test Project")
	project.Mock(now)
	projectRepository := mocks.NewMockProjectRepository(ctrl)
	projectRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Any()).Return(project, nil)

	application := repositories.NewApplication(virtualServer.Id(), project.Id(), "app", "App", repositories.ApplicationTypeConfidential, []string{"http://localhost/callback"})
	application.Mock(now)
	application.SetTokenEndpointAuthMethod(utils.Ptr(method))
	applicationRepository := mocks.NewMockApplicationRepository(ctrl)
	applicationRepository.EXPECT().FirstOrErr(gomock.Any(), gomock.Cond(func(x *repositories.ApplicationFilter) bool {
		return x.GetId() == application.Id()
	})).Return(application, nil)

	applicationKeyRepository := mocks.NewMockApplicationKeyRepository(ctrl)

	dc := ioc.NewDependencyCollection()
	dbContext := mocks2.NewMockContext(ctrl)
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) database.Context {
		return dbContext
	})
	dbContext.EXPECT().VirtualServers().Return(virtualServerRepository).AnyTimes()
	dbContext.EXPECT().Projects().Return(projectRepository).AnyTimes()
	dbContext.EXPECT().Applications().Return(applicationRepository).AnyTimes()
	dbContext.EXPECT().ApplicationKeys().Return(applicationKeyRepository).AnyTimes()

	scope := dc.BuildProvider()
	t.Cleanup(func() {
		utils.PanicOnError(scope.Close, "closing scope")
	})

	return applicationKeyFixture{
		ctx:         middlewares.ContextWithScope(t.Context(), scope),
		application: application,
		keys:        applicationKeyRepository,
	}
}

func (s *AddApplicationKeyCommandSuite) TestHappyPath() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	f := arrangeApplicationKeyFixture(s.T(), ctrl, repositories.TokenEndpointAuthMethodPrivateKeyJwt)
	publicKey := testPublicKeyPem()

	f.keys.EXPECT().FirstOrNil(gomock.Any(), gomock.Cond(func(x *repositories.ApplicationKeyFilter) bool {
		return x.GetApplicationId() == f.application.Id() && x.GetKid() == "key-1"
	})).Return(nil, nil)
	f.keys.EXPECT().Insert(gomock.Cond(func(x *repositories.ApplicationKey) bool {
		return x.ApplicationId() == f.application.Id() && x.Kid() == "key-1" && x.PublicKey() == publicKey
	}))

	// act
	resp, err := HandleAddApplicationKey(f.ctx, AddApplicationKey{
		VirtualServerName: "virtualServer",
		ProjectSlug:       "project",
		ApplicationId:     f.application.Id(),
		Kid:               utils.Ptr("key-1"),
		PublicKey:         publicKey,
	})

	// assert
	s.Require().NoError(err)
	s.Equal("key-1", resp.Kid)
}

func (s *AddApplicationKeyCommandSuite) TestGeneratesKidWhenMissing() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	f := arrangeApplicationKeyFixture(s.T(), ctrl, repositories.TokenEndpointAuthMethodPrivateKeyJwt)
	f.keys.EXPECT().FirstOrNil(gomock.Any(), gomock.Any()).Return(nil, nil)
	f.keys.EXPECT().Insert(gomock.Any())

	// act
	resp, err := HandleAddApplicationKey(f.ctx, AddApplicationKey{
		VirtualServerName: "virtualServer",
		ProjectSlug:       "project",
		ApplicationId:     f.application.Id(),
		PublicKey:         testPublicKeyPem(),
	})

	// assert
	s.Require().NoError(err)
	s.NotEmpty(resp.Kid)
}

func (s *AddApplicationKeyCommandSuite) TestRejectsClientSecretApplication() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	f := arrangeApplicationKeyFixture(s.T(), ctrl, repositories.TokenEndpointAuthMethodClientSecret)

	// act
	_, err := HandleAddApplicationKey(f.ctx, AddApplicationKey{
		VirtualServerName: "virtualServer",
		ProjectSlug:       "project",
		ApplicationId:     f.application.Id(),
		PublicKey:         testPublicKeyPem(),
	})

	// assert
	s.Require().ErrorIs(err, utils.ErrHttpBadRequest)
}

func (s *AddApplicationKeyCommandSuite) TestRejectsInvalidPem() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	f := arrangeApplicationKeyFixture(s.T(), ctrl, repositories.TokenEndpointAuthMethodPrivateKeyJwt)

	// act
	_, err := HandleAddApplicationKey(f.ctx, AddApplicationKey{
		VirtualServerName: "virtualServer",
		ProjectSlug:       "project",
		ApplicationId:     f.application.Id(),
		PublicKey:         "not a key",
	})

	// assert
	s.Require().ErrorIs(err, utils.ErrHttpBadRequest)
}

func (s *AddApplicationKeyCommandSuite) TestRejectsWeakRsaKey() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	f := arrangeApplicationKeyFixture(s.T(), ctrl, repositories.TokenEndpointAuthMethodPrivateKeyJwt)

	// act
	_, err := HandleAddApplicationKey(f.ctx, AddApplicationKey{
		VirtualServerName: "virtualServer",
		ProjectSlug:       "project",
		ApplicationId:     f.application.Id(),
		PublicKey:         testRsaPublicKeyPem(1024),
	})

	// assert
	s.Require().ErrorIs(err, utils.ErrHttpBadRequest)
}

func (s *AddApplicationKeyCommandSuite) TestRejectsDuplicateKid() {
	// arrange
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	f := arrangeApplicationKeyFixture(s.T(), ctrl, repositories.TokenEndpointAuthMethodPrivateKeyJwt)
	f.keys.EXPECT().FirstOrNil(gomock.Any(), gomock.Any()).
		Return(repositories.NewApplicationKey(f.application.Id(), "key-1", testPublicKeyPem()), nil)

	// act
	_, err := HandleAddApplicationKey(f.ctx, AddApplicationKey{
		VirtualServerName: "virtualServer",
		ProjectSlug:       "project",
		ApplicationId:     f.application.Id(),
		Kid:               utils.Ptr("key-1"),
		PublicKey:         testPublicKeyPem(),
	})

	// assert
	s.Require().ErrorIs(err, utils.ErrHttpConflict)
}
