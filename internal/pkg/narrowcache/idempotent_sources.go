package narrowcache

import (
	"context"
	"fmt"
	"sync"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type idempotentSources struct {
	registered map[string]map[string]bool // Map <group name> * <resource key> * <is active>
	mut        sync.RWMutex
}

func ipsKey(strGVK string, obj client.Object) string {
	return fmt.Sprintf("%s|%s|%s", strGVK, obj.GetNamespace(), obj.GetName())
}

// setActive sets the watch as active, and returns true if it already existed (hence not to register again)
func (s *idempotentSources) setActive(group string, strGVK string, obj client.Object) bool {
	k := ipsKey(strGVK, obj)
	s.mut.Lock()
	defer s.mut.Unlock()
	forController, hasGroup := s.registered[group]
	if !hasGroup {
		forController = make(map[string]bool)
		s.registered[group] = forController
	}
	_, exists := forController[k]
	forController[k] = true
	return exists
}

func (s *idempotentSources) isActive(group string, strGVK string, obj client.Object) bool {
	k := ipsKey(strGVK, obj)
	s.mut.RLock()
	defer s.mut.RUnlock()
	if forController, hasGroup := s.registered[group]; hasGroup {
		return forController[k]
	}
	return false
}

func (s *idempotentSources) resetActive(group string) {
	s.mut.Lock()
	defer s.mut.Unlock()
	if forController, hasGroup := s.registered[group]; hasGroup {
		// Reset all registered watches as inactive
		for k := range forController {
			forController[k] = false
		}
	}
}

func (c *Client) ResetActiveWatches(group string) {
	c.idempotentSources.resetActive(group)
}

// SafeEnqueueRequestOnEvents is like EnqueueRequestOnEvents with idempotency: it ensures there is no existing enqueuing callbacks already created for that object.
// If checkActive is false, it creates an always-enabled watch, intended for static resource watching.
// Else, the created watch is ignored when not actively looked-up for in a reconcile loop, so it is intended for dynamic resource watching, such as based on a potentially changing configuration.
func (c *Client) SafeEnqueueRequestOnEvents(ctx context.Context, group string, ctrl controller.Controller, obj client.Object, req reconcile.Request, checkActive bool) error {
	gvk, err := c.GroupVersionKindFor(obj)
	if err != nil {
		return err
	}
	strGVK := gvk.String()
	if _, managed := c.watchedGVKs[strGVK]; managed {
		if exists := c.idempotentSources.setActive(group, strGVK, obj); exists {
			// Do not register again
			return nil
		}

		predicate := func(_ client.Object) bool { return true }
		if checkActive {
			// The watch might be registered, but inactive
			predicate = func(o client.Object) bool {
				return c.idempotentSources.isActive(group, strGVK, o)
			}
		}
		if err := c.EnqueueRequestOnEvents(ctx, ctrl, obj, req, predicate); err != nil {
			// Roll back the reservation so a later reconcile can retry.
			objKey := ipsKey(strGVK, obj)
			c.idempotentSources.mut.Lock()
			delete(c.idempotentSources.registered[group], objKey)
			c.idempotentSources.mut.Unlock()
			return err
		}
		return nil
	}
	return fmt.Errorf("cannot configure IdempotentEnqueueRequestOnEvents on %s: GVK not managed through narrowcache", strGVK)
}
