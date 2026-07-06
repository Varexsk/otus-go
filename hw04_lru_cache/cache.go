package hw04lrucache

import "sync"

type Key string

type Cache interface {
	Set(key Key, value any) bool
	Get(key Key) (any, bool)
	Clear()
}

type lruCache struct {
	capacity int
	queue    List
	items    map[Key]*ListItem
	mu       sync.RWMutex
}

func (l *lruCache) Set(key Key, value any) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	item, ok := l.items[key]
	if ok {
		item.Value = value
		l.queue.MoveToFront(item)
	} else {
		if l.queue.Len() > l.capacity {
			li := l.queue.Back()
			l.queue.Remove(li)
			delete(l.items, li.Key.(Key))
		}
		li := l.queue.PushFront(value, key)
		l.items[key] = li
	}

	return ok
}

func (l *lruCache) Get(key Key) (any, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	item, ok := l.items[key]
	if ok {
		l.queue.MoveToFront(item)
		return item.Value, ok
	}
	return nil, ok
}

func (l *lruCache) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.items = make(map[Key]*ListItem, l.capacity)

	if l.queue.Len() > 0 {
		l.queue.Front().Next = nil
		l.queue.Back().Prev = nil
		l.queue.ResetLen()
	}
}

func NewCache(capacity int) Cache {
	return &lruCache{
		capacity: capacity,
		queue:    NewList(),
		items:    make(map[Key]*ListItem, capacity),
	}
}
