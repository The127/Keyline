package change

// List is a generic type used to track changes in a model.
// T must conform to the comparable constraint to ensure consistent behaviour with map keys.
type List[T comparable] struct {
	changes map[T]struct{}
}

func NewChanges[T comparable]() List[T] {
	return List[T]{
		changes: make(map[T]struct{}),
	}
}

// GetChanges returns a slice of all tracked changes in the Changes collection.
func (b *List[T]) GetChanges() []T {
	changes := make([]T, 0, len(b.changes))
	for change := range b.changes {
		changes = append(changes, change)
	}

	return changes
}

// ClearChanges removes all tracked changes by resetting the internal map to an empty state.
// This method is only supposed to be called by the repository implementations.
func (b *List[T]) ClearChanges() {
	b.changes = make(map[T]struct{})
}

// TrackChange adds the given key to the changes map to indicate a modification of the corresponding property.
func (b *List[T]) TrackChange(key T) {
	b.changes[key] = struct{}{}
}

// HasChanges checks if any changes have been tracked in the Changes collection.
// Returns true if there are tracked changes, false otherwise.
func (b *List[T]) HasChanges() bool {
	return len(b.changes) > 0
}
