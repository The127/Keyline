package utils

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type MapSliceSuite struct {
	suite.Suite
}

func TestMapSliceSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MapSliceSuite))
}

func (s *MapSliceSuite) TestMapsNilToNil() {
	// arrange
	var v []string = nil

	// act
	result := MapSlice(v, func(x string) bool {
		return true
	})

	// assert
	s.Nil(result)
}

func (s *MapSliceSuite) TestMapsEmptyToEmpty() {
	// arrange
	var v = make([]string, 0)

	// act
	result := MapSlice(v, func(x string) bool {
		return true
	})

	// assert
	s.Empty(result)
}

func (s *MapSliceSuite) TestMapsValues() {
	// arrange
	var v = []string{"a", "b", "c"}

	// act
	result := MapSlice(v, func(x string) bool {
		return x == "b"
	})

	// assert
	s.Equal([]bool{false, true, false}, result)
}
