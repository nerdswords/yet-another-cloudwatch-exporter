package job

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompact(t *testing.T) {
	type data struct {
		n int
	}

	type testCase struct {
		name        string
		input       []*data
		keepFunc    func(el *data) bool
		expectedRes []*data
	}

	testCases := []testCase{
		{
			name:        "empty",
			input:       []*data{},
			keepFunc:    nil,
			expectedRes: []*data{},
		},
		{
			name:        "one element input, one element result",
			input:       []*data{{n: 0}},
			keepFunc:    func(el *data) bool { return true },
			expectedRes: []*data{{n: 0}},
		},
		{
			name:        "one element input, empty result",
			input:       []*data{{n: 0}},
			keepFunc:    func(el *data) bool { return false },
			expectedRes: []*data{},
		},
		{
			name:        "two elements input, two elements result",
			input:       []*data{{n: 0}, {n: 1}},
			keepFunc:    func(el *data) bool { return true },
			expectedRes: []*data{{n: 0}, {n: 1}},
		},
		{
			name:        "two elements input, one element result",
			input:       []*data{{n: 0}, {n: 1}},
			keepFunc:    func(el *data) bool { return el.n > 0 },
			expectedRes: []*data{{n: 1}},
		},
		{
			name:        "two elements input, empty result",
			input:       []*data{{n: 0}, {n: 1}},
			keepFunc:    func(el *data) bool { return false },
			expectedRes: []*data{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := compact(tc.input, tc.keepFunc)
			require.Equal(t, tc.expectedRes, res)
		})
	}
}
