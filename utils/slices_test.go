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

type EmptyIfNilSuite struct {
	suite.Suite
}

func TestEmptyIfNilSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(EmptyIfNilSuite))
}

func (s *EmptyIfNilSuite) TestEmptyIfNil() {
	// arrange
	var v []string = nil

	// act
	result := EmptyIfNil(v)

	// assert
	s.Empty(result)
	s.NotNil(result)
	s.Equal([]string{}, result)
}

func (s *EmptyIfNilSuite) TestKeepsOriginalIfNotNil() {
	// arrange
	var v = []string{"a", "b", "c"}

	// act
	result := EmptyIfNil(v)

	// assert
	s.Equal(v, result)
}
