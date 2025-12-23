package toolexec

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestExecutorRace(t *testing.T) {
	registry := NewRegistry()
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("tool-%d", i)
		tool := NewMockTool(name, "desc").WithExecuteFunc(func(ctx context.Context, input *Input) (*Output, error) {
			// Simulate some work and potential for races
			time.Sleep(1 * time.Millisecond)
			return NewOutput().WithMessage("done"), nil
		})
		_ = registry.Register(tool)
	}

	// Use multiple executors sharing the same registry
	exec1 := NewExecutor(registry, WithMaxConcurrent(5))
	exec2 := NewExecutor(registry, WithMaxConcurrent(5))

	const numGoroutines = 50
	const opsPerGoroutine = 20
	var wg sync.WaitGroup

	wg.Add(numGoroutines * 2)

	// Executor 1: Execute synchronous
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				name := fmt.Sprintf("tool-%d", (id+j)%10)
				_, _ = exec1.Execute(context.Background(), name, NewInput())
			}
		}(i)
	}

	// Executor 2: ExecuteMany
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine/5; j++ {
				batch := []ToolExecution{
					{ToolName: "tool-1", Input: NewInput()},
					{ToolName: "tool-2", Input: NewInput()},
					{ToolName: "tool-3", Input: NewInput()},
				}
				_, _ = exec2.ExecuteMany(context.Background(), batch)
			}
		}(i)
	}
	wg.Wait()
}

func TestExecutorExecuteManyStress(t *testing.T) {
	registry := NewRegistry()
	failTool := NewMockTool("fail", "desc").WithExecuteFunc(func(ctx context.Context, input *Input) (*Output, error) {
		return nil, errors.New("failed")
	})
	slowTool := NewMockTool("slow", "desc").WithExecuteFunc(func(ctx context.Context, input *Input) (*Output, error) {
		select {
		case <-time.After(100 * time.Millisecond):
			return NewOutput(), nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})
	_ = registry.Register(failTool)
	_ = registry.Register(slowTool)

	exec := NewExecutor(registry, WithMaxConcurrent(2))

	const numExecs = 50
	executions := make([]ToolExecution, numExecs)
	for i := 0; i < numExecs; i++ {
		if i == 5 { // One fail early
			executions[i] = ToolExecution{ToolName: "fail", Input: NewInput()}
		} else {
			executions[i] = ToolExecution{ToolName: "slow", Input: NewInput()}
		}
	}

	results, err := exec.ExecuteMany(context.Background(), executions)

	if err == nil {
		t.Error("Expected error from ExecuteMany")
	}

	if len(results) != numExecs {
		t.Errorf("Expected %d results, got %d", numExecs, len(results))
	}

	for i, r := range results {
		if r == nil {
			t.Errorf("Result[%d] is nil", i)
			continue
		}
		if r.Error == nil && i == 5 {
			t.Errorf("Result[5] should have errored")
		}
	}
}

func TestExecutorAsyncRace(t *testing.T) {

	registry := NewRegistry()
	tool := NewMockTool("tool", "desc")
	_ = registry.Register(tool)
	exec := NewExecutor(registry)

	const numOps = 100
	var wg sync.WaitGroup
	wg.Add(numOps)

	for i := 0; i < numOps; i++ {
		go func() {
			defer wg.Done()
			ch := exec.ExecuteAsync(context.Background(), "tool", NewInput())
			res := <-ch
			if res == nil {
				t.Error("Result should not be nil")
			}
		}()
	}
	wg.Wait()
}
