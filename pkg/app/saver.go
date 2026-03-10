// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Hans Jørgen Grimstad

package app

import (
	"sync"
	"sync/atomic"

	"citybeestgo/pkg/model"
)

type saveRequest struct {
	path       string
	gene       *model.Gene
	generation int
}

type asyncGeneSaver struct {
	queue    chan saveRequest
	stopCh   chan struct{}
	doneCh   chan struct{}
	closed   atomic.Bool
	stopOnce sync.Once
}

func newAsyncGeneSaver(buffer int) *asyncGeneSaver {
	if buffer <= 0 {
		buffer = 1
	}
	s := &asyncGeneSaver{
		queue:  make(chan saveRequest, buffer),
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
	go s.run()
	return s
}

func (s *asyncGeneSaver) Queue(path string, gene *model.Gene, generation int) {
	if gene == nil || s.closed.Load() {
		return
	}
	req := saveRequest{
		path:       path,
		gene:       gene,
		generation: generation,
	}
	select {
	case s.queue <- req:
	default:
		// Keep caller non-blocking even when queue is full.
		go func() {
			select {
			case s.queue <- req:
			case <-s.stopCh:
			}
		}()
	}
}

func (s *asyncGeneSaver) Stop() {
	s.stopOnce.Do(func() {
		s.closed.Store(true)
		close(s.stopCh)
		<-s.doneCh
	})
}

func (s *asyncGeneSaver) run() {
	defer close(s.doneCh)
	for {
		select {
		case req := <-s.queue:
			_ = req.gene.Save(req.path, req.generation)
		case <-s.stopCh:
			return
		}
	}
}
