package sharedmem

import (
	"fmt"
	"sync"
)

// TaskMemoryTracker tracks allocated memory pages per task.
type TaskMemoryTracker struct {
	mt *MemTable

	AllocLock sync.Mutex
	// Map taskID -> list of allocated regions
	TaskAllocations map[string][]MemRegion
}

// NewTaskMemoryTracker creates a new tracker with given MemTable
func NewTaskMemoryTracker(mt *MemTable) *TaskMemoryTracker {
	return &TaskMemoryTracker{
		mt:              mt,
		TaskAllocations: make(map[string][]MemRegion),
	}
}

// AllocPagesForTask allocates a memory region of size bytes for a task
// and records ownership.
func (t *TaskMemoryTracker) AllocPagesForTask(taskID string, size uint64, owner string) (MemRegion, error) {
	t.AllocLock.Lock()
	defer t.AllocLock.Unlock()

	region, err := t.mt.AllocRegion(size, owner)
	if err != nil {
		return MemRegion{}, fmt.Errorf("allocation failed: %w", err)
	}

	t.TaskAllocations[taskID] = append(t.TaskAllocations[taskID], region)
	return region, nil
}

// FreeTaskPages frees all memory regions allocated for a task.
func (t *TaskMemoryTracker) FreeTaskPages(taskID string) error {
	t.AllocLock.Lock()
	defer t.AllocLock.Unlock()

	regions, ok := t.TaskAllocations[taskID]
	if !ok {
		return fmt.Errorf("no allocations found for task %s", taskID)
	}

	for _, region := range regions {
		err := t.mt.FreeRegion(region.StartAddr)
		if err != nil {
			return fmt.Errorf("failed to free region at 0x%x: %w", region.StartAddr, err)
		}
	}

	delete(t.TaskAllocations, taskID)
	return nil
}

// GetTaskAllocations returns the allocated regions for a task.
func (t *TaskMemoryTracker) GetTaskAllocations(taskID string) ([]MemRegion, bool) {
	t.AllocLock.Lock()
	defer t.AllocLock.Unlock()

	regions, ok := t.TaskAllocations[taskID]
	return regions, ok
}
