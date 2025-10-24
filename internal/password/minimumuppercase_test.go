package password

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMinimumUpperCasePolicy_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		minAmount int
		wantErr   bool
	}{
		{
			name:      "too few upper case letters",
			input:     "FOO",
			minAmount: 5,
			wantErr:   true,
		},
		{
			name:      "just enough upper case letters",
			input:     "FOO",
			minAmount: 3,
			wantErr:   false,
		},
		{
			name:      "dont care (0)",
			input:     "FOO",
			minAmount: 0,
			wantErr:   false,
		},
		{
			name:      "more than enough upper case letters",
			input:     "FOO",
			minAmount: 1,
			wantErr:   false,
		},
		{
			name:      "too few upper case letters (mixed)",
			input:     "FaObO",
			minAmount: 5,
			wantErr:   true,
		},
		{
			name:      "just enough upper case letters (mixed)",
			input:     "FoOOasfasd",
			minAmount: 3,
			wantErr:   false,
		},
		{
			name:      "dont care (0) (mixed)",
			input:     "FOlllaO",
			minAmount: 0,
			wantErr:   false,
		},
		{
			name:      "more than enough upper case letters (mixed)",
			input:     "FabrOO",
			minAmount: 1,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// arrange
			testee := minimumUpperCasePolicy{
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
