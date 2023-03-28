package main

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestYACEApp_FeatureFlagsParsedCorrectly(t *testing.T) {
	app := NewYACEApp()

	// two feature flags
	app.Action = func(c *cli.Context) error {
		featureFlags := c.StringSlice(enableFeatureFlag)
		require.Equal(t, []string{"feature1", "feature2"}, featureFlags)
		return nil
	}

	require.NoError(t, app.Run([]string{"yace", "-enable-feature=feature1,feature2"}), "error running test command")

	// empty feature flags
	app.Action = func(c *cli.Context) error {
		featureFlags := c.StringSlice(enableFeatureFlag)
		require.Len(t, featureFlags, 0)
		return nil
	}

	require.NoError(t, app.Run([]string{"yace"}), "error running test command")
}
