package repositories

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTemplateFilter(t *testing.T) {
	// arrange
	f := NewTemplateFilter()
	var templateType TemplateType = "foo"
	virtualServerId := uuid.New()

	// act
	f = f.TemplateType(templateType)
	f = f.VirtualServerId(virtualServerId)

	// assert
	assert.Equal(t, &templateType, f.templateType)
	assert.Equal(t, &virtualServerId, f.virtualServerId)
}
