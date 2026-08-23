package hw06pipelineexecution

type (
	In  = <-chan interface{}
	Out = In
	Bi  = chan interface{}
)

type Stage func(in In) (out Out)

func ExecutePipeline(in In, done In, stages ...Stage) Out {
	out := in
	for _, stage := range stages {
		out = stage(doneSignal(done, out))
	}
	return out
}

func doneSignal(done In, out Out) Out {
	result := make(Bi)

	go func() {
		defer func() {
			for range out { //nolint:revive
			}
		}()
		defer close(result)

		for {
			select {
			case <-done:
				return
			case v, ok := <-out:
				if !ok {
					return
				}
				select {
				case <-done:
					return
				case result <- v:
				}
			}
		}
	}()
	return result
}
