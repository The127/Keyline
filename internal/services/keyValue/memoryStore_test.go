package keyValue

import (
	"Keyline/internal/clock"
	"Keyline/internal/middlewares"
	"Keyline/ioc"
	"Keyline/utils"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type MemoryStoreSuite struct {
	suite.Suite
}

func TestMemoryStoreSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MemoryStoreSuite))
}

func (s *MemoryStoreSuite) createContext() (context.Context, clock.TimeSetterFn) {
	dc := ioc.NewDependencyCollection()

	clockService, timeSetter := clock.NewMockServiceNow()
	ioc.RegisterTransient(dc, func(dp *ioc.DependencyProvider) clock.Service {
		return clockService
	})

	scope := dc.BuildProvider()
	s.T().Cleanup(func() {
		utils.PanicOnError(scope.Close, "closing scope")
	})

	return middlewares.ContextWithScope(s.T().Context(), scope), timeSetter
}

func (s *MemoryStoreSuite) TestSetGet() {
	// arrange
	ctx, _ := s.createContext()
	store := NewMemoryStore()

	// act
	err := store.Set(ctx, "key", "value")
	s.Require().NoError(err)

	got, err := store.Get(ctx, "key")
	s.Require().NoError(err)

	// assert
	s.Equal("value", got)
}

func (s *MemoryStoreSuite) TestSetDeleteGet() {
	// arrange
	ctx, _ := s.createContext()
	store := NewMemoryStore()

	// act
	err := store.Set(ctx, "key", "value")
	s.Require().NoError(err)

	err = store.Delete(ctx, "key")
	s.Require().NoError(err)

	got, err := store.Get(ctx, "key")

	// assert
	s.Equal(ErrNotFound, err)
	s.Empty(got)
}

func (s *MemoryStoreSuite) TestGetExpired() {
	// arrange
	ctx, setTime := s.createContext()
	store := NewMemoryStore()

	// act
	err := store.Set(ctx, "key", "value", WithExpiration(time.Second))
	s.Require().NoError(err)

	setTime(time.Now().Add(time.Second * 2))

	got, err := store.Get(ctx, "key")

	// assert
	s.Equal(ErrNotFound, err)
	s.Empty(got)
}
