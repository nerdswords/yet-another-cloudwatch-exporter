package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFeatureFlagsInContext_DefaultsToNonEnabled(t *testing.T) {
	flags := FlagsFromCtx(context.Background())
	require.False(t, flags.IsFeatureEnabled("some-feature"))
	require.False(t, flags.IsFeatureEnabled("some-other-feature"))
}

type flags struct{}

func (f flags) IsFeatureEnabled(_ string) bool {
	return true
}

func TestFeatureFlagsInContext_RetrievesFlagsFromContext(t *testing.T) {
	ctx := CtxWithFlags(context.Background(), flags{})
	require.True(t, FlagsFromCtx(ctx).IsFeatureEnabled("some-feature"))
	require.True(t, FlagsFromCtx(ctx).IsFeatureEnabled("some-other-feature"))
}
