package utils

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
)

type UnwrapSuite struct {
	suite.Suite
}

func TestUnwrapSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UnwrapSuite))
}

func (s *UnwrapSuite) TestNoErrorReturnsValue() {
	// arrange
	t := "asd"
	var err error = nil

	// act
	unwrapped := Unwrap(t, err)

	// assert
	s.Equal(t, unwrapped, t)
}

func (s *UnwrapSuite) TestPanicsOnError() {
	// arrange
	t := ""
	var err error = errors.New("error")

	// act
	s.Panics(func() {
		_ = Unwrap(t, err)
	})
}
