package utils

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"
)

type TypeOfSuite struct {
	suite.Suite
}

func TestTypeOfSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(TypeOfSuite))
}

func (s *TypeOfSuite) TestBool() {
	// act
	result := TypeOf[bool]()

	// assert
	s.Equal(reflect.TypeOf(true), result)
}

func (s *TypeOfSuite) TestString() {
	// act
	result := TypeOf[string]()

	// assert
	s.Equal(reflect.TypeOf(""), result)
}

type re struct {
	Field string
}

func (s *TypeOfSuite) TestStruct() {
	// act
	result := TypeOf[re]()

	// assert
	s.Equal(reflect.TypeOf(re{}), result)
}

func (s *TypeOfSuite) TestPointer() {
	// act
	result := TypeOf[*string]()

	// assert
	val := ""
	s.Equal(reflect.TypeOf(&val), result)
}

func (s *TypeOfSuite) TestMap() {
	// act
	result := TypeOf[map[string]string]()

	// assert
	s.Equal(reflect.TypeOf(map[string]string{}), result)
}

func (s *TypeOfSuite) TestSlice() {
	// act
	result := TypeOf[[]string]()

	// assert
	s.Equal(reflect.TypeOf([]string{}), result)
}
