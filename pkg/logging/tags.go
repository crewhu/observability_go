package logging

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
)

type Tags map[string]any
type key int

const tagsKey key = iota

func WithTag(ctx context.Context, key string, value any) context.Context {
	// getTags returns the map by reference: without cloning, sibling goroutines
	// derived from the same parent ctx would mutate the same map (cross-handler
	// tag leakage plus a concurrent map write panic under -race).
	tags := maps.Clone(getTags(ctx))
	tags[key] = value
	return context.WithValue(ctx, tagsKey, tags)
}

func WithTags(ctx context.Context, tags Tags) context.Context {
	t := maps.Clone(getTags(ctx))
	maps.Copy(t, tags)
	return context.WithValue(ctx, tagsKey, t)
}

func getTags(ctx context.Context) Tags {
	tags := ctx.Value(tagsKey)
	if tags == nil {
		return make(Tags)
	}
	return tags.(Tags)
}

// mergeCtxTags merges the ctx-accumulated tags with call-site extra tags,
// extra winning on a key collision. Returns the ctx map as-is (read-only
// here) when there is nothing to merge, to avoid an allocation on the
// common no-call-site-tags path.
func mergeCtxTags(ctx context.Context, extra Tags) Tags {
	if len(extra) == 0 {
		return getTags(ctx)
	}
	return getTags(ctx).Merge(extra)
}

func (ts Tags) Merge(t Tags) Tags {
	n := Tags{}
	maps.Copy(n, ts)
	maps.Copy(n, t)
	return n
}

func (ts Tags) String() string {
	var r []string
	for key, value := range ts {
		r = append(r, fmt.Sprintf("[%s:%+v]", key, value))
	}
	slices.Sort(r)
	return strings.Join(r, " ")
}
