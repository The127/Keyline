package utils

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type MapPtrSuite struct {
	suite.Suite
}

func TestMapPtrSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MapPtrSuite))
}

func (s *MapPtrSuite) TestMapsNilToNil() {
	// arrange
	var v *string = nil

	// act
	result := MapPtr(v, func(x string) bool {
		return true
	})

	// assert
	s.Nil(result)
}

func (s *MapPtrSuite) TestMapsIfNotNil() {
	// arrange
	var v = "not nil"

	// act
	result := MapPtr(&v, func(x string) bool {
		return true
	})

	// assert
	s.Require().NotNil(result)
	s.True(*result)
}

type NilIfZeroSuite struct {
	suite.Suite
}

func TestNilIfZeroSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(NilIfZeroSuite))
}

func (s *NilIfZeroSuite) TestReturnsNilIfZeroValue() {
	// arrange
	var v = 0

	// act
	result := NilIfZero(v)

	// assert
	s.Nil(result)
}

func (s *NilIfZeroSuite) TestReturnsValueIfNotZero() {
	// arrange
	var v = 1

	// act
	result := NilIfZero(v)

	// assert
	s.NotNil(result)
	s.Equal(1, *result)
}

type ZeroIfNilSuite struct {
	suite.Suite
}

func TestZeroIfNilSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ZeroIfNilSuite))
}

func (s *ZeroIfNilSuite) TestReturnsZeroIfNil() {
	// arrange
	var v *int = nil

	// act
	result := ZeroIfNil(v)

	// assert
	s.Equal(0, result)
}

func (s *ZeroIfNilSuite) TestReturnsValueIfNotZero() {
	// arrange
	var v = 1

	// act
	result := ZeroIfNil(&v)

	// assert
	s.Equal(1, result)
}
