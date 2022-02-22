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
		{configFile: "multiple_roles.ok.yml"},
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

func TestDimensionLabelField(t *testing.T) {
	var testCases = []struct {
		configFile    string
		presenceCheck bool
	}{
		{configFile: "config_test.yml", presenceCheck: true},
		{configFile: "empty_rolearn.ok.yml", presenceCheck: false},
	}
	for _, tc := range testCases {
		config := ScrapeConf{}
		configFile := fmt.Sprintf("testdata/%s", tc.configFile)
		if err := config.Load(&configFile); err != nil {
			t.Error(err)
			t.FailNow()
		}
		var keyPrefix *string = config.DimensionLabelPrefix
		if tc.presenceCheck {
			if keyPrefix == nil {
				t.Error("key 'dimensionLabelPrefix' was not parsed successfully.")
			}
		} else {
			if keyPrefix != nil {
				t.Error("key 'dimensionLabelPrefix' was not expected to be set")
			}
		}
	}
}
