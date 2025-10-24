package password

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMinimumNumbersPolicy_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		minAmount int
		wantErr   bool
	}{
		{
			name:      "too few numbers letters",
			input:     "123",
			minAmount: 5,
			wantErr:   true,
		},
		{
			name:      "just enough numbers letters",
			input:     "456",
			minAmount: 3,
			wantErr:   false,
		},
		{
			name:      "dont care (0)",
			input:     "789",
			minAmount: 0,
			wantErr:   false,
		},
		{
			name:      "more than enough numbers letters",
			input:     "000",
			minAmount: 1,
			wantErr:   false,
		},
		{
			name:      "too few numbers letters (mixed)",
			input:     "F1a2O3!bO",
			minAmount: 5,
			wantErr:   true,
		},
		{
			name:      "just enough numbers letters (mixed)",
			input:     "Fo4O5O1!as!fa!sd",
			minAmount: 3,
			wantErr:   false,
		},
		{
			name:      "dont care (0) (mixed)",
			input:     "FOl2l1!4laO",
			minAmount: 0,
			wantErr:   false,
		},
		{
			name:      "more than enough numbers letters (mixed)",
			input:     "Fa5!b7r9O!O",
			minAmount: 1,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// arrange
			testee := minimumNumbersPolicy{
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
