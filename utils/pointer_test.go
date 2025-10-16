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
