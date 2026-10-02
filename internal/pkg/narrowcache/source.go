package narrowcache

import (
	"context"
	"errors"

	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// EventFilter receives the previous and current object for an event. Create events have a nil previous object,
// delete events have a nil current object, and update events have both values populated.
type EventFilter func(oldObject, newObject client.Object) bool

type NarrowSource struct {
	source.Source
	handler handler.EventHandler
	onStart func(ctx context.Context, q workqueue.TypedRateLimitingInterface[reconcile.Request]) error
}

func (s *NarrowSource) Start(ctx context.Context, q workqueue.TypedRateLimitingInterface[reconcile.Request]) error {
	if s.handler == nil {
		return errors.New("must specify NarrowSource.handler")
	}
	return s.onStart(ctx, q)
}

type requestEventHandler struct {
	request reconcile.Request
	filter  EventFilter
}

func (h requestEventHandler) Create(_ context.Context, e event.CreateEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	h.enqueue(nil, e.Object, q)
}

func (h requestEventHandler) Update(_ context.Context, e event.UpdateEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	h.enqueue(e.ObjectOld, e.ObjectNew, q)
}

func (h requestEventHandler) Delete(_ context.Context, e event.DeleteEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	h.enqueue(e.Object, nil, q)
}

func (h requestEventHandler) Generic(_ context.Context, e event.GenericEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	h.enqueue(nil, e.Object, q)
}

func (h requestEventHandler) enqueue(oldObject, newObject client.Object, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	if h.filter == nil || h.filter(oldObject, newObject) {
		q.Add(h.request)
	}
}
