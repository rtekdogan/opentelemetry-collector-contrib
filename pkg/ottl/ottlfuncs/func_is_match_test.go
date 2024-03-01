// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ottlfuncs

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
)

func Test_isMatch(t *testing.T) {
	tests := []struct {
		name     string
		target   ottl.StringLikeGetter[any]
		pattern  ottl.StringLikeGetter[any]
		expected bool
	}{
		{
			name: "replace match true",
			target: &ottl.StandardStringLikeGetter[any]{
				Getter: func(ctx context.Context, tCtx any) (any, error) {
					return "hello world", nil
				},
			},

			pattern: &ottl.StandardStringLikeGetter[any]{
				Getter: func(ctx context.Context, tCtx any) (any, error) {
					return "hello.*", nil
				},
			},
			expected: true,
		},
		{
			name: "replace match false",
			target: &ottl.StandardStringLikeGetter[any]{
				Getter: func(ctx context.Context, tCtx any) (any, error) {
					return "goodbye world", nil
				},
			},
			pattern: &ottl.StandardStringLikeGetter[any]{
				Getter: func(ctx context.Context, tCtx any) (any, error) {
					return "hello.*", nil
				},
			},
			expected: false,
		},
		{
			name: "replace match complex",
			target: &ottl.StandardStringLikeGetter[any]{
				Getter: func(ctx context.Context, tCtx any) (any, error) {
					return "-12.001", nil
				},
			},
			pattern: &ottl.StandardStringLikeGetter[any]{
				Getter: func(ctx context.Context, tCtx any) (any, error) {
					return "[-+]?\\d*\\.\\d+([eE][-+]?\\d+)?", nil
				},
			},
			expected: true,
		},
		{
			name: "target bool",
			target: &ottl.StandardStringLikeGetter[any]{
				Getter: func(ctx context.Context, tCtx any) (any, error) {
					return true, nil
				},
			},
			pattern: &ottl.StandardStringLikeGetter[any]{
				Getter: func(ctx context.Context, tCtx any) (any, error) {
					return "true", nil
				},
			},
			expected: true,
		},
		{
			name: "target int",
			target: &ottl.StandardStringLikeGetter[any]{
				Getter: func(ctx context.Context, tCtx any) (any, error) {
					return int64(1), nil
				},
			},
			pattern: &ottl.StandardStringLikeGetter[any]{
				Getter: func(ctx context.Context, tCtx any) (any, error) {
					return "\\d", nil
				},
			},
			expected: true,
		},
		{
			name: "target float",
			target: &ottl.StandardStringLikeGetter[any]{
				Getter: func(ctx context.Context, tCtx any) (any, error) {
					return 1.1, nil
				},
			},
			pattern: &ottl.StandardStringLikeGetter[any]{
				Getter: func(ctx context.Context, tCtx any) (any, error) {
					return "\\d.\\d", nil
				},
			},
			expected: true,
		},
		{
			name: "target pcommon.Value",
			target: &ottl.StandardStringLikeGetter[any]{
				Getter: func(ctx context.Context, tCtx any) (any, error) {
					v := pcommon.NewValueEmpty()
					v.SetStr("test")
					return v, nil
				},
			},
			pattern: &ottl.StandardStringLikeGetter[any]{
				Getter: func(ctx context.Context, tCtx any) (any, error) {
					return "test", nil
				},
			},
			expected: true,
		},
		{
			name: "nil target",
			target: &ottl.StandardStringLikeGetter[any]{
				Getter: func(ctx context.Context, tCtx any) (any, error) {
					return nil, nil
				},
			},
			pattern: &ottl.StandardStringLikeGetter[any]{
				Getter: func(ctx context.Context, tCtx any) (any, error) {
					return "impossible to match", nil
				},
			},
			expected: false,
		},
		{
			name: "nil pattern",
			target: &ottl.StandardStringLikeGetter[any]{
				Getter: func(ctx context.Context, tCtx any) (any, error) {
					return "hello world", nil
				},
			},
			pattern: &ottl.StandardStringLikeGetter[any]{
				Getter: func(ctx context.Context, tCtx any) (any, error) {
					return nil, nil
				},
			},
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exprFunc, err := isMatch(tt.target, tt.pattern)
			assert.NoError(t, err)
			result, err := exprFunc(context.Background(), nil)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func Test_isMatch_validation(t *testing.T) {
	target := &ottl.StandardStringLikeGetter[any]{
		Getter: func(ctx context.Context, tCtx any) (any, error) {
			return "anything", nil
		},
	}
	pattern := &ottl.StandardStringLikeGetter[any]{
		Getter: func(ctx context.Context, tCtx any) (any, error) {
			return "\\K", nil
		},
	}
	f, err := isMatch[any](target, pattern)
	assert.NoError(t, err)
	_, err = f(context.Background(), nil)
	require.Error(t, err)
}

func Test_isMatch_error(t *testing.T) {
	target := &ottl.StandardStringLikeGetter[any]{
		Getter: func(ctx context.Context, tCtx any) (any, error) {
			return make(chan int), nil
		},
	}
	pattern := &ottl.StandardStringLikeGetter[any]{
		Getter: func(ctx context.Context, tCtx any) (any, error) {
			return "test", nil
		},
	}
	exprFunc, err := isMatch[any](target, pattern)
	assert.NoError(t, err)
	_, err = exprFunc(context.Background(), nil)
	require.Error(t, err)
}

func Benchmark_performance_same_pattern(b *testing.B) {
	target := &ottl.StandardStringLikeGetter[any]{
		Getter: func(ctx context.Context, tCtx any) (any, error) {
			return "abcde", nil
		},
	}
	pattern := &ottl.StandardStringLikeGetter[any]{
		Getter: func(ctx context.Context, tCtx any) (any, error) {
			return "a.*", nil
		},
	}
	for i := 0; i < b.N; i++ {
		exprFunx, err := isMatch[any](target, pattern)
		assert.NoError(b, err)
		match, err := exprFunx(context.Background(), nil)
		assert.NoError(b, err)
		assert.Equal(b, match, true)
	}
}

func Benchmark_performance_different_pattern(b *testing.B) {
	target := &ottl.StandardStringLikeGetter[any]{
		Getter: func(ctx context.Context, tCtx any) (any, error) {
			return "abcd", nil
		},
	}

	for i := 0; i < b.N; i++ {
		pattern := &ottl.StandardStringLikeGetter[any]{
			Getter: func(ctx context.Context, tCtx any) (any, error) {
				return fmt.Sprint(i, "a*"), nil
			},
		}
		exprFunx, err := isMatch[any](target, pattern)
		assert.NoError(b, err)
		match, err := exprFunx(context.Background(), nil)
		assert.NoError(b, err)
		assert.Equal(b, match, false)
	}
}
