package pagination_test

import (
	"testing"

	"github.com/ship-monitor/cloud/internal/v2/domain/pagination"
	"github.com/stretchr/testify/require"
)

func TestSize(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		limit, want int
		invalid     bool
	}{
		{0, pagination.DefaultLimit, false},
		{1, 1, false},
		{pagination.MaxLimit, pagination.MaxLimit, false},
		{-1, 0, true},
		{pagination.MaxLimit + 1, 0, true},
	} {
		size, err := (pagination.Request[int]{
			After: nil,
			Limit: testCase.limit,
		}).Size()
		if testCase.invalid {
			require.ErrorIs(t, err, pagination.ErrInvalidLimit)
		} else {
			require.NoError(t, err)
		}

		require.Equal(t, testCase.want, size)
	}
}

func TestFromRows(t *testing.T) {
	t.Parallel()

	key := func(v int) int { return v }
	page := pagination.FromRows([]int{1, 2, 3}, 2, key)
	require.Equal(t, []int{1, 2}, page.Items)
	require.Equal(t, 2, *page.Next)

	last := pagination.FromRows([]int{3}, 2, key)
	require.Equal(t, []int{3}, last.Items)
	require.Nil(t, last.Next)
}
