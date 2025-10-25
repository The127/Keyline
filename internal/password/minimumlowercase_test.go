package password

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMinimumLowerCasePolicy_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		minAmount int
		wantErr   bool
	}{
		{
			name:      "too few lower case letters",
			input:     "foo",
			minAmount: 5,
			wantErr:   true,
		},
		{
			name:      "just enough lower case letters",
			input:     "foo",
			minAmount: 3,
			wantErr:   false,
		},
		{
			name:      "dont care (0)",
			input:     "foo",
			minAmount: 0,
			wantErr:   false,
		},
		{
			name:      "more than enough lower case letters",
			input:     "foo",
			minAmount: 1,
			wantErr:   false,
		},
		{
			name:      "too few lower case letters (mixed)",
			input:     "fooB",
			minAmount: 5,
			wantErr:   true,
		},
		{
			name:      "just enough lower case letters (mixed)",
			input:     "foBARo",
			minAmount: 3,
			wantErr:   false,
		},
		{
			name:      "dont care (0) (mixed)",
			input:     "foKLOAOo",
			minAmount: 0,
			wantErr:   false,
		},
		{
			name:      "more than enough lower case letters (mixed)",
			input:     "foFo",
			minAmount: 1,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// arrange
			testee := minimumLowerCasePolicy{
				MinAmount: tt.minAmount,
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
