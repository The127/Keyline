package password

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLengthPolicy_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     string
		minLength int
		wantErr   bool
	}{
		{
			name:      "too short",
			input:     "foo",
			minLength: 5,
			wantErr:   true,
		},
		{
			name:      "not too short",
			input:     "foo",
			minLength: 1,
			wantErr:   false,
		},
		{
			name:      "zero min length",
			input:     "foo",
			minLength: 0,
			wantErr:   false,
		},
		{
			name:      "zero empty string",
			input:     "",
			minLength: 0,
			wantErr:   false,
		},
		{
			name:      "non zero empty string",
			input:     "",
			minLength: 1,
			wantErr:   true,
		},
		{
			name:      "just right",
			input:     "foo",
			minLength: 3,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// arrange
			testee := minLengthPolicy{
				minLength: tt.minLength,
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
