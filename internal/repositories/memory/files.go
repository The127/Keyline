package memory

import (
	"github.com/The127/Keyline/internal/change"
	"github.com/The127/Keyline/internal/repositories"
	"github.com/The127/Keyline/utils"
	"context"
	"sync"

	"github.com/google/uuid"
)

type FileRepository struct {
	store         map[uuid.UUID]*repositories.File
	mu            *sync.RWMutex
	changeTracker *change.Tracker
	entityType    int
}

func NewFileRepository(store map[uuid.UUID]*repositories.File, mu *sync.RWMutex, changeTracker *change.Tracker, entityType int) *FileRepository {
	return &FileRepository{
		store:         store,
		mu:            mu,
		changeTracker: changeTracker,
		entityType:    entityType,
	}
}

func (r *FileRepository) matches(f *repositories.File, filter *repositories.FileFilter) bool {
	if filter.HasId() && f.Id() != filter.GetId() {
		return false
	}
	return true
}

func (r *FileRepository) FirstOrErr(ctx context.Context, filter *repositories.FileFilter) (*repositories.File, error) {
	result, err := r.FirstOrNil(ctx, filter)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, utils.ErrFileNotFoud
	}
	return result, nil
}

func (r *FileRepository) FirstOrNil(_ context.Context, filter *repositories.FileFilter) (*repositories.File, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, f := range r.store {
		if r.matches(f, filter) {
			return f, nil
		}
	}
	return nil, nil
}

func (r *FileRepository) Insert(file *repositories.File) {
	r.changeTracker.Add(change.NewEntry(change.Added, r.entityType, file))
}
