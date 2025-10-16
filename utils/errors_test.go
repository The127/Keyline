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
	s.Equal(t, unwrapped)
}

func (s *UnwrapSuite) TestPanicsOnError() {
	// arrange
	t := ""
	var err = errors.New("error")

	// act
	s.Panics(func() {
		_ = Unwrap(t, err)
	})
}

type PanicOnErrorSuite struct {
	suite.Suite
}

func TestPanicOnErrorSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(PanicOnErrorSuite))
}

func (s *PanicOnErrorSuite) TestPanicsOnError() {
	// arrange
	var err = errors.New("error")

	// act
	s.Panics(func() {
		PanicOnError(func() error {
			return err
		}, "message")
	})
}

func (s *PanicOnErrorSuite) TestDoesNotPanicWhenNoError() {
	// arrange
	var err error = nil

	// act
	s.NotPanics(func() {
		PanicOnError(func() error {
			return err
		}, "message")
	})
}
