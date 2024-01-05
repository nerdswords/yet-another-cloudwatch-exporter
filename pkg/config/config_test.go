package config

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
)

func TestConfLoad(t *testing.T) {
	testCases := []struct {
		configFile string
	}{
		{configFile: "config_test.yml"},
		{configFile: "empty_rolearn.ok.yml"},
		{configFile: "sts_region.ok.yml"},
		{configFile: "multiple_roles.ok.yml"},
		{configFile: "custom_namespace.ok.yml"},
	}
	for _, tc := range testCases {
		config := ScrapeConf{}
		configFile := fmt.Sprintf("testdata/%s", tc.configFile)
		if _, err := config.Load(configFile, logging.NewNopLogger()); err != nil {
			t.Error(err)
			t.FailNow()
		}
	}
}

func TestBadConfigs(t *testing.T) {
	testCases := []struct {
		configFile string
		errorMsg   string
	}{
		{
			configFile: "externalid_without_rolearn.bad.yml",
			errorMsg:   "RoleArn should not be empty",
		},
		{
			configFile: "externalid_with_empty_rolearn.bad.yml",
			errorMsg:   "RoleArn should not be empty",
		},
		{
			configFile: "unknown_version.bad.yml",
			errorMsg:   "unknown apiVersion value 'invalidVersion'",
		},
		{
			configFile: "custom_namespace_without_name.bad.yml",
			errorMsg:   "Name should not be empty",
		},
		{
			configFile: "custom_namespace_without_namespace.bad.yml",
			errorMsg:   "Namespace should not be empty",
		},
		{
			configFile: "custom_namespace_without_region.bad.yml",
			errorMsg:   "Regions should not be empty",
		},
	}

	for _, tc := range testCases {
		config := ScrapeConf{}
		configFile := fmt.Sprintf("testdata/%s", tc.configFile)
		if _, err := config.Load(configFile, logging.NewNopLogger()); err != nil {
			if !strings.Contains(err.Error(), tc.errorMsg) {
				t.Errorf("expecter error for config file %q to contain %q but got: %s", tc.configFile, tc.errorMsg, err)
				t.FailNow()
			}
		} else {
			t.Log("expected validation error")
			t.FailNow()
		}
	}
}

func TestValidateConfigFailuresWhenUsingAsLibrary(t *testing.T) {
	type testcase struct {
		config   ScrapeConf
		errorMsg string
	}
	testCases := map[string]testcase{
		"empty role should be configured when environment role is desired": {
			config: ScrapeConf{
				APIVersion: "v1alpha1",
				StsRegion:  "us-east-2",
				Discovery: Discovery{
					Jobs: []*Job{{
						Regions: []string{"us-east-2"},
						Type:    "sqs",
						Metrics: []*Metric{{
							Name:       "NumberOfMessagesSent",
							Statistics: []string{"Average"},
						}},
					}},
				},
			},
			errorMsg: "no IAM roles configured. If the current IAM role is desired, an empty Role should be configured",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			_, err := tc.config.Validate()
			require.Error(t, err, "Expected config validation to fail")
			require.Equal(t, tc.errorMsg, err.Error())
		})
	}
}
