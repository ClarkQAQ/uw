package uboot

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"

	"uw/ulog"
	"uw/umap"
)

const (
	eventQueueSize   = 1024
	eventQueueWorker = 12
)

var (
	event                 = umap.NewHmap[string, *umap.Hmap[string, EventHandler[any]]]()
	eventSubscribeRWMutex = &sync.RWMutex{}
	eventQueue            = make([]*eventQueueData, 0, eventQueueSize)
	eventQueueMutex       = &sync.Mutex{}
	eventCond             = sync.NewCond(&sync.Mutex{})
)

type EventKey struct {
	string
}

func (ek EventKey) String() string {
	return ek.string
}

func NewEventKey(value string) EventKey {
	return EventKey{
		string: value,
	}
}

type eventQueueData struct {
	Key     EventKey
	Data    any
	OnError func(error)
}

type EventHandler[T any] func(ctx context.Context, data T) error

func popEventQueue() *eventQueueData {
	eventQueueMutex.Lock()
	defer eventQueueMutex.Unlock()

	if len(eventQueue) < 1 {
		return nil
	}

	n := len(eventQueue)
	x := eventQueue[0]
	eventQueue = eventQueue[1:n]

	return x
}

func eventWorkerJob() bool {
	if data := popEventQueue(); data != nil {
		if e := Publish(data.Key, data.Data); e != nil && data.OnError != nil {
			ulog.Warn("event worker publish %s: %s", data.Key, e)
			data.OnError(e)
			return true
		}
	}

	return false
}

func eventWorker() {
	for {
		if eventWorkerJob() {
			continue
		}

		eventCond.L.Lock()
		eventCond.Wait()
		eventCond.L.Unlock()
	}
}

func init() {
	for i := 0; i < eventQueueWorker; i++ {
		go eventWorker()
	}
}

func generateKey(h *umap.Hmap[string, EventHandler[any]], maxRetries int) (string, error) {
	for i := 0; i < maxRetries; i++ {
		b := make([]byte, 64)
		if _, e := rand.Read(b); e != nil {
			return "", e
		}
		hh := sha256.New()
		if _, e := hh.Write(b); e != nil {
			return "", e
		}

		key := hex.EncodeToString(hh.Sum(nil))

		if _, ok := h.Load(key); !ok {
			return key, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique key, retries %d", maxRetries)
}

func Subscribe[T any](key EventKey, handler EventHandler[T]) (string, error) {
	eventSubscribeRWMutex.Lock()
	defer eventSubscribeRWMutex.Unlock()

	if _, ok := event.Load(key.String()); !ok {
		event.Set(key.String(), umap.NewHmap[string, EventHandler[any]]())
	}

	eh := event.Get(key.String())

	kk, e := generateKey(eh, 12)
	if e != nil {
		return "", e
	}

	eh.Set(kk, func(ctx context.Context, data any) error {
		if d, ok := data.(T); ok {
			return handler(ctx, d)
		}

		return nil
	})

	return kk, nil
}

func Publish[T any](key EventKey, data T) error {
	eventSubscribeRWMutex.RLock()
	defer eventSubscribeRWMutex.RUnlock()

	ctx, errorsPool := context.Background(), error(nil)

	if v, ok := event.Load(key.String()); ok {
		v.Range(func(handlerKey string, handler EventHandler[any]) bool {
			defer func() {
				if r := recover(); r != nil {
					errorsPool = errors.Join(errorsPool, fmt.Errorf("recover: %v", r))
				}
			}()

			if e := handler(ctx, data); e != nil {
				errorsPool = errors.Join(errorsPool, e)
			}

			return true
		})

		return errorsPool
	}

	return errors.New("event not found")
}

func PublishQueue[T any](key EventKey, data T, onError func(error)) {
	eventQueueMutex.Lock()
	defer eventQueueMutex.Unlock()

	eventQueue = append(eventQueue, &eventQueueData{
		Key:     key,
		Data:    data,
		OnError: onError,
	})

	eventCond.Signal()
}

func UnSubscribe(key EventKey, handlerKey ...string) error {
	eventSubscribeRWMutex.Lock()
	defer eventSubscribeRWMutex.Unlock()

	if v, ok := event.Load(key.String()); ok {
		if len(handlerKey) < 1 {
			event.Delete(key.String())
			return nil
		}

		for _, k := range handlerKey {
			v.Delete(k)
		}
	}

	return nil
}
