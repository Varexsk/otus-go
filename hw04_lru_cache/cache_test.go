package hw04lrucache

import (
	"math/rand"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCache(t *testing.T) {
	t.Run("empty cache", func(t *testing.T) {
		c := NewCache(10)

		_, ok := c.Get("aaa")
		require.False(t, ok)

		_, ok = c.Get("bbb")
		require.False(t, ok)
	})

	t.Run("simple", func(t *testing.T) {
		c := NewCache(5)

		wasInCache := c.Set("aaa", 100)
		require.False(t, wasInCache)

		wasInCache = c.Set("bbb", 200)
		require.False(t, wasInCache)

		val, ok := c.Get("aaa")
		require.True(t, ok)
		require.Equal(t, 100, val)

		val, ok = c.Get("bbb")
		require.True(t, ok)
		require.Equal(t, 200, val)

		wasInCache = c.Set("aaa", 300)
		require.True(t, wasInCache)

		val, ok = c.Get("aaa")
		require.True(t, ok)
		require.Equal(t, 300, val)

		val, ok = c.Get("ccc")
		require.False(t, ok)
		require.Nil(t, val)
	})

	t.Run("clear logic", func(t *testing.T) {
		c := NewCache(5)
		c.Set(Key("0"), 0)
		c.Set(Key("1"), 1)
		c.Set(Key("2"), 2)
		c.Set(Key("3"), 3)
		c.Set(Key("4"), 4)

		c.Clear()

		for i := range 5 {
			v, ok := c.Get(Key(strconv.Itoa(i)))
			require.Equal(t, false, ok)
			require.Nil(t, v)
		}
	})

	t.Run("purge logic", func(t *testing.T) {
		c := NewCache(3)

		c.Set(Key("0"), 0)
		c.Set(Key("1"), 1)
		c.Set(Key("2"), 2)

		c.Set(Key("3"), 3)

		_, ok := c.Get(Key("0"))
		require.Equal(t, false, ok)

		_, ok = c.Get(Key("1"))
		require.Equal(t, true, ok)

		_, ok = c.Get(Key("2"))
		require.Equal(t, true, ok)

		_, ok = c.Get(Key("3"))
		require.Equal(t, true, ok)
	})

	t.Run("purge logic with access", func(t *testing.T) {
		c := NewCache(3)

		c.Set(Key("0"), 0)
		c.Set(Key("1"), 1)
		c.Set(Key("2"), 2)

		c.Get(Key("0"))
		c.Get(Key("1"))	

		c.Set(Key("3"), 3)

		_, ok := c.Get(Key("0"))
		require.Equal(t, true, ok)

		_, ok = c.Get(Key("1"))
		require.Equal(t, true, ok)

		_, ok = c.Get(Key("2"))
		require.Equal(t, false, ok)
	})
}

func TestCacheMultithreading(t *testing.T) {
	_ = t

	c := NewCache(10)
	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 1_000_000; i++ {
			c.Set(Key(strconv.Itoa(i)), i)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 1_000_000; i++ {
			c.Get(Key(strconv.Itoa(rand.Intn(1_000_000))))
		}
	}()

	wg.Wait()
}
