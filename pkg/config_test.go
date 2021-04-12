package exporter

import (
	"testing"
)

func TestConfLoad(t *testing.T) {
	config := ScrapeConf{}
	configFile := "config_test.yml"
	if err := config.Load(&configFile); err != nil {
		t.Error(err)
	}
}
