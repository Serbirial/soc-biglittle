package sharedmem

import (
	"fmt"
	"sync"
)

// TaskMemoryTracker tracks allocated memory pages per task.
type TaskMemoryTracker struct {
	mt *MemTable

	allocLock sync.Mutex
	// Map taskID -> list of allocated regions
	taskAllocations map[string][]MemRegion
}

// NewTaskMemoryTracker creates a new tracker with given MemTable
func NewTaskMemoryTracker(mt *MemTable) *TaskMemoryTracker {
	return &TaskMemoryTracker{
		mt:              mt,
		taskAllocations: make(map[string][]MemRegion),
	}
}

// AllocPagesForTask allocates a memory region of size bytes for a task
// and records ownership.
func (t *TaskMemoryTracker) AllocPagesForTask(taskID string, size uint64, owner string) (MemRegion, error) {
	t.allocLock.Lock()
	defer t.allocLock.Unlock()

	region, err := t.mt.AllocRegion(size, owner)
	if err != nil {
		return MemRegion{}, fmt.Errorf("allocation failed: %w", err)
	}

	t.taskAllocations[taskID] = append(t.taskAllocations[taskID], region)
	return region, nil
}

// FreeTaskPages frees all memory regions allocated for a task.
func (t *TaskMemoryTracker) FreeTaskPages(taskID string) error {
	t.allocLock.Lock()
	defer t.allocLock.Unlock()

	regions, ok := t.taskAllocations[taskID]
	if !ok {
		return fmt.Errorf("no allocations found for task %s", taskID)
	}

	for _, region := range regions {
		err := t.mt.FreeRegion(region.StartAddr)
		if err != nil {
			return fmt.Errorf("failed to free region at 0x%x: %w", region.StartAddr, err)
		}
	}

	delete(t.taskAllocations, taskID)
	return nil
}

// GetTaskAllocations returns the allocated regions for a task.
func (t *TaskMemoryTracker) GetTaskAllocations(taskID string) ([]MemRegion, bool) {
	t.allocLock.Lock()
	defer t.allocLock.Unlock()

	regions, ok := t.taskAllocations[taskID]
	return regions, ok
}
