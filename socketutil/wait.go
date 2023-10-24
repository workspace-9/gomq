package socketutil

import "context"

type WaitCloser[T any] struct {
	ctx    context.Context
	cancel context.CancelFunc
	done   chan T
}

func NewWaitCloser[T any](ctx context.Context) WaitCloser[T] {
	derived, cancel := context.WithCancel(ctx)
	return WaitCloser[T]{
		ctx: derived, cancel: cancel, done: make(chan T),
	}
}

func (w WaitCloser[T]) Close() T {
	w.cancel()
	return <-w.done
}

func (w WaitCloser[T]) Done() <-chan struct{} {
	return w.ctx.Done()
}

func (w WaitCloser[T]) Finish(t T) {
	w.done <- t
}
