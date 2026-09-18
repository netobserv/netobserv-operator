package enqueuer

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Intended for tracking static resources with always-enabled watch
type Static interface {
	EnqueueOnChange(context.Context, client.Object, reconcile.Request) error
}

// Intended for tracking dynamic resources, tracking the watch status (active/inactive)
type Dynamic interface {
	Static
	ResetActiveWatches()
}
