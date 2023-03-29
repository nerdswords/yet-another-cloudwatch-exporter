package config

import "context"

const (
	// EncodingResourceAssociator is a feature flag used to toggle to the new resource association algorithm, implemented in discovery mode. See
	// https://github.com/nerdswords/yet-another-cloudwatch-exporter/pull/833 for more details.
	EncodingResourceAssociator = "encoding-resource-associator"
)

var (
	flagsCtxKey       = struct{}{}
	defaultController = noController{}
)

// FeatureFlags is an interface all objects that can tell wether or not a feature flag is enabled can implement.
type FeatureFlags interface {
	// IsFeatureEnabled tells if the feature flag identified by flag is enabled.
	IsFeatureEnabled(flag string) bool
}

// CtxWithFlags injects a FeatureFlags inside a given context.Context, so that they are easily communicated through layers.
func CtxWithFlags(ctx context.Context, ctrl FeatureFlags) context.Context {
	return context.WithValue(ctx, flagsCtxKey, ctrl)
}

// FlagsFromCtx retrieves a FeatureFlags from a given context.Context, defaulting to one with all feature flags disabled if none is found.
func FlagsFromCtx(ctx context.Context) FeatureFlags {
	if ctrl := ctx.Value(flagsCtxKey); ctrl != nil {
		return ctrl.(FeatureFlags)
	}
	return defaultController
}

// noController implements a no-op FeatureFlags
type noController struct{}

func (nc noController) IsFeatureEnabled(_ string) bool {
	return false
}
