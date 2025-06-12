package app

import (
	"context"
	"sync"
)

// TODO: Can this be updated to a simple Run() function like Ollama?
type Lifecycle struct {
	Ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewLifecycle() *Lifecycle {
	ctx, cancel := context.WithCancel(context.Background())
	return &Lifecycle{Ctx: ctx, cancel: cancel}
}

func (l *Lifecycle) Start() {
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		// Run background tasks, e.g., keep-alive ping, logging
		<-l.Ctx.Done()
	}()
}

func (l *Lifecycle) Shutdown() {
	l.cancel()
	l.wg.Wait()
}
