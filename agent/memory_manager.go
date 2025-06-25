package agent

import (
	"context"
	"fmt"
	"sync"

	"bigLITTLE/sharedmem"
)

type TaskMemoryManager struct {
	memMgr  *MemoryManager
	tracker *sharedmem.TaskMemoryTracker

	lock sync.Mutex
}

func NewTaskMemoryManager(memMgr *MemoryManager, tracker *sharedmem.TaskMemoryTracker) *TaskMemoryManager {
	return &TaskMemoryManager{
		memMgr:  memMgr,
		tracker: tracker,
	}
}

// Alloc allocates size bytes for a task and records allocation.
func (t *TaskMemoryManager) Alloc(taskID string, size uint64, owner string) (sharedmem.MemRegion, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	region, err := t.memMgr.Table.AllocRegion(size, owner)
	if err != nil {
		return sharedmem.MemRegion{}, fmt.Errorf("alloc failed: %w", err)
	}
	// Record allocation in tracker
	t.tracker.AllocLock.Lock()
	t.tracker.TaskAllocations[taskID] = append(t.tracker.TaskAllocations[taskID], region)
	t.tracker.AllocLock.Unlock()

	return region, nil
}

// FreeTask frees all pages allocated to a task.
func (t *TaskMemoryManager) FreeTask(taskID string) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.tracker.AllocLock.Lock()
	regions, ok := t.tracker.TaskAllocations[taskID]
	t.tracker.AllocLock.Unlock()
	if !ok {
		return fmt.Errorf("no allocations for task %s", taskID)
	}

	for _, r := range regions {
		err := t.memMgr.FreeRegion(r.StartAddr)
		if err != nil {
			return fmt.Errorf("failed freeing region 0x%x: %w", r.StartAddr, err)
		}
	}

	t.tracker.AllocLock.Lock()
	delete(t.tracker.TaskAllocations, taskID)
	t.tracker.AllocLock.Unlock()

	return nil
}

// Read, Write, etc can just delegate to underlying MemoryManager
func (t *TaskMemoryManager) Read(ctx context.Context, addr uint64, size uint64) ([]byte, error) {
	return t.memMgr.Read(ctx, addr, size)
}

func (t *TaskMemoryManager) Write(ctx context.Context, addr uint64, data []byte) error {
	return t.memMgr.Write(ctx, addr, data)
}
