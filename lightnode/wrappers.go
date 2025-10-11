package lightnode

import (
	"context"
	"fmt"

	"github.com/anyproto/any-sync/app"
)

// noOpComponent provides no-op lifecycle methods for shared components.
// When embedded in a wrapper, it prevents re-initialization of already-running components.
type noOpComponent struct {
	name string
}

func (n *noOpComponent) Name() string                  { return n.name }
func (n *noOpComponent) Init(_ *app.App) error         { return nil }
func (n *noOpComponent) Run(_ context.Context) error   { return nil }
func (n *noOpComponent) Close(_ context.Context) error { return nil }

// sharedComponent is a generic wrapper that prevents re-initialization of shared components.
// It embeds noOpComponent for lifecycle no-ops and the actual component interface.
type sharedComponent[T any] struct {
	noOpComponent

	component T
}

// newSharedComponent fetches a typed component from the app, wraps it, and prevents re-initialization.
func newSharedComponent[T any](appInstance *app.App, name string) *sharedComponent[T] {
	comp := appInstance.MustComponent(name)

	typed, ok := comp.(T)
	if !ok {
		panic(fmt.Sprintf("component %s has unexpected type %T", name, comp))
	}

	return &sharedComponent[T]{
		noOpComponent: noOpComponent{name: name},
		component:     typed,
	}
}

// Name returns the component name, overriding any Name() method in the wrapped component.
func (s *sharedComponent[T]) Name() string {
	return s.noOpComponent.Name()
}

// Unwrap returns the underlying component for type-safe access.
func (s *sharedComponent[T]) Unwrap() T {
	return s.component
}
