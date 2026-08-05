package hw04lrucache

type List interface {
	Len() int
	ResetLen()
	Front() *ListItem
	Back() *ListItem
	PushFront(v interface{}) *ListItem
	PushBack(v interface{}) *ListItem
	Remove(i *ListItem)
	MoveToFront(i *ListItem)
}

type ListItem struct {
	Value interface{}
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

func (l *list) PushFront(v interface{}) *ListItem {
	l.len++

	li := &ListItem{
		Value: v,
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

func (l *list) PushBack(v interface{}) *ListItem {
	l.len++

	li := &ListItem{
		Value: v,
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
	if li.Prev != nil {
		li.Prev.Next = li.Next
	} else {
		l.head = li.Next
	}
	if li.Next != nil {
		li.Next.Prev = li.Prev
	} else {
		l.tail = li.Prev
	}
	li.Next, li.Prev = nil, nil
	l.len--
}


func (l *list) MoveToFront(li *ListItem) {
	if li == l.head {
		return
	}
	l.Remove(li)
	li.Next = l.head
	l.head.Prev = li
	l.head = li
	l.len++
}

func NewList() List {
	return new(list)
}
