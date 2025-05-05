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
	tags := getTags(ctx)
	tags[key] = value
	return context.WithValue(ctx, tagsKey, tags)
}

func WithTags(ctx context.Context, tags Tags) context.Context {
	t := getTags(ctx)
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
