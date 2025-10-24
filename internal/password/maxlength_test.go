package password

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMaxLengthPolicy_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     string
		maxLength int
		wantErr   bool
	}{
		{
			name:      "too long",
			input:     "foo",
			maxLength: 2,
			wantErr:   true,
		},
		{
			name:      "just right",
			input:     "foo",
			maxLength: 3,
			wantErr:   false,
		},
		{
			name:      "unicode length",
			input:     "😶‍🌫️",
			maxLength: 1,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// arrange
			testee := maxLengthPolicy{
				maxLength: tt.maxLength,
			}

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
