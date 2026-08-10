package hw05parallelexecution

import (
	"errors"
	"sync"
	"sync/atomic"
)

var ErrErrorsLimitExceeded = errors.New("errors limit exceeded")

type Task func() error

// Run starts tasks in n goroutines and stops its work when receiving m errors from tasks.
func Run(tasks []Task, n, m int) error {
	if n <= 0 {
		return nil
	}

	if m <= 0 {
		return ErrErrorsLimitExceeded
	}

	taskChan := make(chan Task, len(tasks))
	var errCount atomic.Int32
	wg := &sync.WaitGroup{}

	limitExceeded := func() bool {
		return int(errCount.Load()) >= m
	}

	for _, task := range tasks {
		taskChan <- task
	}

	close(taskChan)

	wg.Add(n)
	for range n {
		go func() {
			defer wg.Done()
			for task := range taskChan {
				if limitExceeded() {
					return
				}
				if err := task(); err != nil {
					errCount.Add(1)
				}
			}
		}()
	}

	wg.Wait()

	if limitExceeded() {
		return ErrErrorsLimitExceeded
	}

	return nil
}
