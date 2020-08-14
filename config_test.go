package main

import (
	"testing"
)

func TestConfLoad(t *testing.T) {
	config = conf{}
	configFile := "config_test.yml"
	if err := config.load(&configFile); err != nil {
		t.Error(err)
	}
}
