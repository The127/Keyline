package utils

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ZeroSuite struct {
	suite.Suite
}

func TestZeroSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ZeroSuite))
}

func (s *ZeroSuite) TestPrimitive() {
	// act
	zero := Zero[int]()

	// assert
	s.Equal(0, zero)
}

func (s *ZeroSuite) TestString() {
	// act
	zero := Zero[string]()

	// assert
	s.Empty(zero)
}

func (s *ZeroSuite) TestStruct() {
	// act
	zero := Zero[struct {
		Field string
	}]()

	// assert
	s.Equal(struct {
		Field string
	}{}, zero)
}
