package pool

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// item — тестовый объект, фиксирующий вызовы Reset.
type item struct {
	value  int
	resets int
}

func (i *item) Reset() {
	i.value = 0
	i.resets++
}

func TestNew(t *testing.T) {
	t.Run("returns_pool_pointer", func(t *testing.T) {
		p := New(func() *item { return &item{} })
		require.NotNil(t, p)
	})

	t.Run("infers_type_from_factory", func(t *testing.T) {
		// Компилируется без явного указания [*item] — тип выведен из newFn.
		var p *Pool[*item] = New(func() *item { return &item{} })
		require.NotNil(t, p)
	})
}

func TestPool_Get(t *testing.T) {
	t.Run("creates_object_via_factory_when_empty", func(t *testing.T) {
		calls := 0
		p := New(func() *item {
			calls++
			return &item{value: 42}
		})

		got := p.Get()

		require.NotNil(t, got)
		require.Equal(t, 42, got.value)
		require.Equal(t, 1, calls)
	})

	t.Run("returns_distinct_objects_without_put", func(t *testing.T) {
		p := New(func() *item { return &item{} })

		a := p.Get()
		b := p.Get()

		require.NotSame(t, a, b)
	})
}

func TestPool_Put(t *testing.T) {
	t.Run("calls_reset_before_returning_to_pool", func(t *testing.T) {
		p := New(func() *item { return &item{} })

		obj := p.Get()
		obj.value = 7
		p.Put(obj)

		require.Equal(t, 1, obj.resets)
		require.Equal(t, 0, obj.value)
	})

	t.Run("object_from_get_after_put_is_reset", func(t *testing.T) {
		p := New(func() *item { return &item{} })

		obj := p.Get()
		obj.value = 7
		p.Put(obj)

		// sync.Pool не гарантирует возврат того же объекта, но любой
		// полученный объект обязан быть в сброшенном состоянии.
		got := p.Get()
		require.Equal(t, 0, got.value)
	})
}

func TestPool_Concurrent(t *testing.T) {
	const (
		goroutines = 16
		iterations = 1000
	)

	p := New(func() *item { return &item{} })

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			for range iterations {
				obj := p.Get()
				require.Equal(t, 0, obj.value)
				obj.value++
				p.Put(obj)
			}
		}()
	}
	wg.Wait()
}
