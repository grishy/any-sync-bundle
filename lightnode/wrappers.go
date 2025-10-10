package lightnode

import (
	"context"

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

// newSharedComponent creates a wrapper around a component that prevents re-initialization.
func newSharedComponent[T any](name string, comp T) *sharedComponent[T] {
	return &sharedComponent[T]{
		noOpComponent: noOpComponent{name: name},
		component:     comp,
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
