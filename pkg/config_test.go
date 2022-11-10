package exporter

import (
	"fmt"
	"strings"
	"testing"
)

func TestConfLoad(t *testing.T) {
	var testCases = []struct {
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
		if err := config.Load(&configFile); err != nil {
			t.Error(err)
			t.FailNow()
		}
	}
}

func TestBadConfigs(t *testing.T) {
	var testCases = []struct {
		configFile string
		errorMsg   string
	}{
		{
			configFile: "externalid_without_rolearn.bad.yml",
			errorMsg:   "RoleArn should not be empty",
		}, {
			configFile: "externalid_with_empty_rolearn.bad.yml",
			errorMsg:   "RoleArn should not be empty",
		}, {
			configFile: "unknown_version.bad.yml",
			errorMsg:   "apiVersion line missing or version is unknown (invalidVersion)",
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
		if err := config.Load(&configFile); err != nil {
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
