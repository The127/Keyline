package utils

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type JsonMergePatchSuite struct {
	suite.Suite
}

func TestJsonMergePatchSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(JsonMergePatchSuite))
}

func (s *JsonMergePatchSuite) TestAddsToObject() {
	// arrange
	base := map[string]interface{}{
		"oldKey": "oldValue",
	}
	patch := map[string]interface{}{
		"newKey": "newValue",
	}

	// act
	result := JsonMergePatch(base, patch)

	// assert
	s.Len(result, 2)

	s.Contains(result, "oldKey")
	s.Equal("oldValue", result["oldKey"])

	s.Contains(result, "newKey")
	s.Equal("newValue", result["newKey"])
}

func (s *JsonMergePatchSuite) TestRemovesFromObject() {
	// arrange
	base := map[string]interface{}{
		"oldKey": "oldValue",
	}
	patch := map[string]interface{}{
		"oldKey": nil,
	}

	// act
	result := JsonMergePatch(base, patch)

	// assert
	s.Empty(result)
}
