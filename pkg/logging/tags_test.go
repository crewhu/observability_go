package logging

import (
	"context"
	"sync"
	"testing"
)

func TestWithTagDoesNotMutateParentTags(t *testing.T) {
	parent := WithTag(context.Background(), "handler", "ParentHandler")

	_ = WithTag(parent, "extra", "child-only")

	parentTags := getTags(parent)
	if _, ok := parentTags["extra"]; ok {
		t.Fatalf("parent ctx tags mutated by child WithTag call: %+v", parentTags)
	}
	if parentTags["handler"] != "ParentHandler" {
		t.Fatalf("parent ctx lost its own tag: %+v", parentTags)
	}
}

func TestWithTagsDoesNotMutateParentTags(t *testing.T) {
	parent := WithTags(context.Background(), Tags{"handler": "ParentHandler"})

	_ = WithTags(parent, Tags{"extra": "child-only"})

	parentTags := getTags(parent)
	if _, ok := parentTags["extra"]; ok {
		t.Fatalf("parent ctx tags mutated by child WithTags call: %+v", parentTags)
	}
}

// TestWithTagConcurrentSiblingsDoNotRace reproduces the ConnectWise
// tickets/projects orchestrator fan-out (sibling handler goroutines derived
// from the same parent ctx tagging concurrently) that caused cross-handler
// log contamination. Pre-fix, getTags(ctx) returned the map by reference and
// every goroutine wrote into the same map: run with -race and it either
// reports a data race or the runtime aborts with "concurrent map writes".
func TestWithTagConcurrentSiblingsDoNotRace(t *testing.T) {
	const n = 100
	parent := WithTag(context.Background(), "seed", "root")

	var wg sync.WaitGroup
	seen := make([]Tags, n)
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ctx := WithTag(parent, "id", i)
			ctx = WithTags(ctx, Tags{"squared": i * i})
			seen[i] = getTags(ctx)
		}(i)
	}
	wg.Wait()

	for i, tags := range seen {
		if tags["id"] != i {
			t.Fatalf("goroutine %d: got id=%v, want %d (cross-goroutine contamination)", i, tags["id"], i)
		}
		if tags["squared"] != i*i {
			t.Fatalf("goroutine %d: got squared=%v, want %d", i, tags["squared"], i*i)
		}
		if tags["seed"] != "root" {
			t.Fatalf("goroutine %d: lost ancestor tag: %+v", i, tags)
		}
	}

	if _, ok := getTags(parent)["id"]; ok {
		t.Fatalf("parent ctx mutated by child goroutines: %+v", getTags(parent))
	}
}

// TestWithTagsConcurrentSiblingsDoNotRace is the WithTags counterpart of
// TestWithTagConcurrentSiblingsDoNotRace: it calls WithTags directly on the
// shared parent ctx from every goroutine (not on an already goroutine-private
// derived ctx), so it actually stresses WithTags's own clone against the
// shared map instead of a copy WithTag already made private.
func TestWithTagsConcurrentSiblingsDoNotRace(t *testing.T) {
	const n = 100
	parent := WithTags(context.Background(), Tags{"seed": "root"})

	var wg sync.WaitGroup
	seen := make([]Tags, n)
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ctx := WithTags(parent, Tags{"id": i, "squared": i * i})
			seen[i] = getTags(ctx)
		}(i)
	}
	wg.Wait()

	for i, tags := range seen {
		if tags["id"] != i {
			t.Fatalf("goroutine %d: got id=%v, want %d (cross-goroutine contamination)", i, tags["id"], i)
		}
		if tags["squared"] != i*i {
			t.Fatalf("goroutine %d: got squared=%v, want %d", i, tags["squared"], i*i)
		}
		if tags["seed"] != "root" {
			t.Fatalf("goroutine %d: lost ancestor tag: %+v", i, tags)
		}
	}

	if _, ok := getTags(parent)["id"]; ok {
		t.Fatalf("parent ctx mutated by child goroutines: %+v", getTags(parent))
	}
}
