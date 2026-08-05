package hw04lrucache

import "sync"

type Key string

type Cache interface {
	Set(key Key, value interface{}) bool
	Get(key Key) (interface{}, bool)
	Clear()
}

type lruCache struct {
	capacity int
	queue    List
	items    map[Key]*ListItem
	mu       sync.RWMutex
}

type CacheItem struct {
	Value interface{}
	Key   Key
}

func (l *lruCache) Set(key Key, value interface{}) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	item, ok := l.items[key]
	if ok {
		item.Value = CacheItem{
			Value: value,
			Key:   key,
		}
		l.queue.MoveToFront(item)
	} else {
		if l.queue.Len() >= l.capacity {
			li := l.queue.Back()
			l.queue.Remove(li)
			delete(l.items, li.Value.(CacheItem).Key)
		}

		ci := CacheItem{
			Value: value,
			Key:   key,
		}

		li := l.queue.PushFront(ci)
		l.items[key] = li
	}

	return ok
}

func (l *lruCache) Get(key Key) (interface{}, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	item, ok := l.items[key]
	if ok {
		l.queue.MoveToFront(item)
		return item.Value.(CacheItem).Value, ok
	}
	return nil, ok
}

func (l *lruCache) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.items = make(map[Key]*ListItem, l.capacity)
	l.queue = NewList()
}

func NewCache(capacity int) Cache {
	return &lruCache{
		capacity: capacity,
		queue:    NewList(),
		items:    make(map[Key]*ListItem, capacity),
	}
}
