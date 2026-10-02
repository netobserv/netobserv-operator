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

// FilteredStatic can register optional, GVK-supported named watches before a resource is fetched.
type FilteredStatic interface {
	Static
	EnqueueOnChangeIfManaged(context.Context, client.Object, reconcile.Request, func(client.Object, client.Object) bool) error
}

// Intended for tracking dynamic resources, tracking the watch status (active/inactive)
type Dynamic interface {
	Static
	ResetActiveWatches()
}

// FilteredDynamic registers name-scoped watches whose event handling is activated per reconcile.
type FilteredDynamic interface {
	FilteredStatic
	Dynamic
}
