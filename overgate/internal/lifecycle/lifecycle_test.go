package lifecycle

import (
	"context"
	"errors"
	"testing"
	"time"
)

type serviceFunc func(context.Context) error

func (f serviceFunc) Run(ctx context.Context) error { return f(ctx) }

func waitResult(t *testing.T, group *Group) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- group.Wait() }()
	select {
	case err := <-done:
		return err
	case <-time.After(3 * time.Second):
		t.Fatal("Wait blocked")
		return nil
	}
}

func TestEmptyGroup(t *testing.T) {
	if err := waitResult(t, Start(context.Background(), nil)); err != nil {
		t.Fatal(err)
	}
}

func TestCompletionStopsSiblings(t *testing.T) {
	for _, fails := range []bool{false, true} {
		name := "normal_completion"
		if fails {
			name = "failure"
		}
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			firstErr, siblingErr := errors.New("first"), errors.New("sibling")
			started := make(chan struct{})
			stopped := make(chan struct{})
			group := Start(ctx, []LifeCycle{
				serviceFunc(func(context.Context) error {
					<-started
					if fails {
						return firstErr
					}
					return nil
				}),
				serviceFunc(func(ctx context.Context) error {
					close(started)
					<-ctx.Done()
					close(stopped)
					return siblingErr
				}),
			})
			err := waitResult(t, group)
			if !errors.Is(err, siblingErr) || errors.Is(err, firstErr) != fails {
				t.Fatalf("unexpected joined errors: %v", err)
			}
			select {
			case <-stopped:
			default:
				t.Fatal("Wait returned before sibling stopped")
			}
			if ctx.Err() != nil {
				t.Fatal("parent context was canceled")
			}
		})
	}
}

func TestParentCancellation(t *testing.T) {
	for _, count := range []int{1, 3} {
		ctx, cancel := context.WithCancel(context.Background())
		services := make([]LifeCycle, count)
		started := make(chan struct{}, count)
		for i := range services {
			services[i] = serviceFunc(func(ctx context.Context) error {
				started <- struct{}{}
				<-ctx.Done()
				return nil
			})
		}
		group := Start(ctx, services)
		for i := 0; i < count; i++ {
			select {
			case <-started:
			case <-time.After(3 * time.Second):
				cancel()
				t.Fatal("service did not start")
			}
		}
		cancel()
		if err := waitResult(t, group); err != nil {
			t.Fatal(err)
		}
	}
}
