package hooks

import (
	"errors"
	"sync"
)

var (
	ErrUninitialized = errors.New("store_empty")
)

type Result struct {
	Data interface{}
	Err  error
}

type EventHandler func(args ...interface{}) Result

type Event struct {
	Name string
	Fn   EventHandler
}

type EventStore struct {
	mu        *sync.RWMutex
	observers map[string]Event
}

func NewEventStore() *EventStore {
	return &EventStore{
		mu:        &sync.RWMutex{},
		observers: make(map[string]Event),
	}
}

func (e *EventStore) On(name string, fn EventHandler) {
	var store *EventStore

	if e.observers == nil {
		store = NewEventStore()
	} else {
		store = e
	}

	event := Event{
		Name: name,
		Fn:   fn,
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	store.observers[name] = event
}

func (e *EventStore) Emit(event string, data ...interface{}) Result {
	if e.observers == nil {
		return Result{Err: ErrUninitialized}
	}

	e.mu.Lock()
	handler, ok := e.observers[event]
	if !ok {
		e.mu.Unlock()
		return Result{}
	}

	e.mu.Unlock()
	return handler.Fn(data...)
}

func (e *EventStore) Off(event string) {
	if e.observers == nil {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	_, ok := e.observers[event]
	if ok {
		delete(e.observers, event)
	}
}
