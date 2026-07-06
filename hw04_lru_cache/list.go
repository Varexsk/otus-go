package hw04lrucache

type List interface {
	Len() int
	ResetLen()
	Front() *ListItem
	Back() *ListItem
	PushFront(v ...any) *ListItem
	PushBack(v ...any) *ListItem
	Remove(i *ListItem)
	MoveToFront(i *ListItem)
}

type ListItem struct {
	Value any
	Key   any
	Next  *ListItem
	Prev  *ListItem
}

type list struct {
	head *ListItem
	tail *ListItem
	len  int
}

func (l *list) ResetLen() {
	l.len = 0
}

func (l *list) Len() int {
	return l.len
}

func (l *list) Front() *ListItem {
	return l.head
}

func (l *list) Back() *ListItem {
	return l.tail
}

func (l *list) PushFront(v ...any) *ListItem {
	l.len++

	li := &ListItem{
		Value: v[0],
	}

	if len(v) > 1 {
		li.Key = v[1]
	}

	if l.head == nil {
		l.head = li
		l.tail = li
		return li
	}

	li.Next = l.head
	l.head.Prev = li
	l.head = li
	return li
}

func (l *list) PushBack(v ...any) *ListItem {
	l.len++

	li := &ListItem{
		Value: v[0],
	}

	if len(v) > 1 {
		li.Key = v[1]
	}

	if l.tail == nil {
		l.head = li
		l.tail = li
		return li
	}

	li.Prev = l.tail
	l.tail.Next = li
	l.tail = li
	return li
}

func (l *list) Remove(li *ListItem) {
	if li == l.head {
		l.head = li.Next
		li.Next = nil
		l.len--
		return
	}
	if li == l.tail {
		l.tail = li.Prev
		li.Prev = nil
		l.len--
		return
	}

	// Соединяем соседние элементы друг с другом
	li.Prev.Next = li.Next
	li.Next.Prev = li.Prev

	l.len--
}

func (l *list) MoveToFront(li *ListItem) {
	if li == l.head {
		return
	}
	if li == l.tail {
		// Помечаем предыдущий элемент последним
		li.Prev.Next = nil
		l.tail = li.Prev

		// Перемещаем элемент в начало
		li.Prev = nil
		li.Next = l.head
		l.head.Prev = li
		l.head = li
		return
	}

	// Соединяем соседние элементы друг с другом
	li.Prev.Next = li.Next
	li.Next.Prev = li.Prev

	// Перемещаем элемент в начало
	li.Prev = nil
	li.Next = l.head
	l.head.Prev = li
	l.head = li
}

func NewList() List {
	return new(list)
}
