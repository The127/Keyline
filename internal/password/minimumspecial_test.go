package password

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMinimumSpecialPolicy_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		minAmount int
		wantErr   bool
	}{
		{
			name:      "unicode doesn't count (just enough)",
			input:     "💀ьΩ密",
			minAmount: 1,
			wantErr:   true,
		},
		{
			name:      "too few special letters",
			input:     "?=!",
			minAmount: 5,
			wantErr:   true,
		},
		{
			name:      "just enough special letters",
			input:     "())",
			minAmount: 3,
			wantErr:   false,
		},
		{
			name:      "dont care (0)",
			input:     "!!!",
			minAmount: 0,
			wantErr:   false,
		},
		{
			name:      "more than enough special letters",
			input:     "!!!",
			minAmount: 1,
			wantErr:   false,
		},
		{
			name:      "too few special letters (mixed)",
			input:     "FaO!bO",
			minAmount: 5,
			wantErr:   true,
		},
		{
			name:      "just enough special letters (mixed)",
			input:     "FoOO!as!fa!sd",
			minAmount: 3,
			wantErr:   false,
		},
		{
			name:      "dont care (0) (mixed)",
			input:     "FOll!laO",
			minAmount: 0,
			wantErr:   false,
		},
		{
			name:      "more than enough special letters (mixed)",
			input:     "Fa!brO!O",
			minAmount: 1,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// arrange
			testee := minimumSpecialPolicy{
				minAmount: tt.minAmount,
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
