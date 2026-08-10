package hw05parallelexecution

import (
	"errors"
	"fmt"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestRun(t *testing.T) {
	defer goleak.VerifyNone(t)

	t.Run("if were errors in first M tasks, than finished not more N+M tasks", func(t *testing.T) {
		tasksCount := 50
		tasks := make([]Task, 0, tasksCount)

		var runTasksCount int32

		for i := 0; i < tasksCount; i++ {
			err := fmt.Errorf("error from task %d", i)
			tasks = append(tasks, func() error {
				time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)))
				atomic.AddInt32(&runTasksCount, 1)
				return err
			})
		}

		workersCount := 10
		maxErrorsCount := 23
		err := Run(tasks, workersCount, maxErrorsCount)

		require.Truef(t, errors.Is(err, ErrErrorsLimitExceeded), "actual err - %v", err)
		require.LessOrEqual(t, runTasksCount, int32(workersCount+maxErrorsCount), "extra tasks were started")
	})

	t.Run("tasks without errors", func(t *testing.T) {
		tasksCount := 50
		tasks := make([]Task, 0, tasksCount)

		var runTasksCount int32
		var sumTime time.Duration

		for i := 0; i < tasksCount; i++ {
			taskSleep := time.Millisecond * time.Duration(rand.Intn(100))
			sumTime += taskSleep

			tasks = append(tasks, func() error {
				time.Sleep(taskSleep)
				atomic.AddInt32(&runTasksCount, 1)
				return nil
			})
		}

		workersCount := 5
		maxErrorsCount := 1

		start := time.Now()
		err := Run(tasks, workersCount, maxErrorsCount)
		elapsedTime := time.Since(start)
		require.NoError(t, err)

		require.Equal(t, int32(tasksCount), runTasksCount, "not all tasks were completed")
		require.LessOrEqual(t, int64(elapsedTime), int64(sumTime/2), "tasks were run sequentially?")
	})
}

func TestRunConcurrency(t *testing.T) {
	defer goleak.VerifyNone(t)

	const (
		tasksCount   = 50
		workersCount = 5
		waitFor      = 5 * time.Second
		tick         = time.Millisecond
	)

	var activeCount, maxActiveCount, doneCount atomic.Int32

	release := make(chan struct{})

	tasks := make([]Task, 0, tasksCount)
	for range tasksCount {
		tasks = append(tasks, func() error {
			active := activeCount.Add(1)
			for {
				observed := maxActiveCount.Load()
				if active <= observed || maxActiveCount.CompareAndSwap(observed, active) {
					break
				}
			}

			<-release

			activeCount.Add(-1)
			doneCount.Add(1)

			return nil
		})
	}

	var runErr error
	runFinished := make(chan struct{})

	go func() {
		defer close(runFinished)
		runErr = Run(tasks, workersCount, 1)
	}()

	require.Eventually(t, func() bool {
		return activeCount.Load() == int32(workersCount)
	}, waitFor, tick, "expected %d tasks to run simultaneously", workersCount)

	close(release)

	require.Eventually(t, func() bool {
		select {
		case <-runFinished:
			return true
		default:
			return false
		}
	}, waitFor, tick, "Run did not return after all tasks were released")

	require.NoError(t, runErr)
	require.Equal(t, int32(tasksCount), doneCount.Load(), "not all tasks were completed")
	require.LessOrEqual(t, maxActiveCount.Load(), int32(workersCount), "more than n tasks were run simultaneously")
}
