package password

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommonPolicy_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "not in list",
			input:   ":not-in-list:",
			wantErr: false,
		},
		{
			name:    "in list",
			input:   "gondor",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// arrange
			testee := commonPolicy{}

			// act
			err := testee.Validate(tt.input)

			// assert
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
