package dbresolver

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPolicyResolver_RoundRobin(t *testing.T) {
	t.Run("when resource count is 1", func(t *testing.T) {
		rr := NewRoundRobalancer(1)
		require.Equal(t, rr.Get(), int64(0))
		require.Equal(t, rr.Get(), int64(0))
		require.Equal(t, rr.Get(), int64(0))
	})

	t.Run("when resource count is 2", func(t *testing.T) {
		rr := NewRoundRobalancer(2)
		require.Equal(t, rr.Get(), int64(0))
		require.Equal(t, rr.Get(), int64(1))
		require.Equal(t, rr.Get(), int64(0))
		require.Equal(t, rr.Get(), int64(1))
	})

	t.Run("when resource count is 3", func(t *testing.T) {
		rr := NewRoundRobalancer(3)
		require.Equal(t, rr.Get(), int64(0))
		require.Equal(t, rr.Get(), int64(1))
		require.Equal(t, rr.Get(), int64(2))
		require.Equal(t, rr.Get(), int64(0))
		require.Equal(t, rr.Get(), int64(1))
		require.Equal(t, rr.Get(), int64(2))
	})

	t.Run("resource access in parallel", func(t *testing.T) {
		rr := NewRoundRobalancer(3)

		idxs := make(chan int64, 6)
		for i := 0; i < 6; i++ {
			go func() {
				idxs <- rr.Get()
			}()
		}

		got := []int64{}
		for i := 0; i < 6; i++ {
			got = append(got, <-idxs)
		}

		expected := []int64{0, 1, 2, 0, 1, 2}
		require.Equal(t, expected, got)
	})
}
