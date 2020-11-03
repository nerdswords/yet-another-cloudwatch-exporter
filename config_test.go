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

func TestConfLoadAddUntaggedUnsupported(t *testing.T) {
	config = conf{}
	configFile := "config_test_adduntagged_non_lambda.yml"
	err := config.load(&configFile)

	if err == nil {
		t.Error("Invalid addUntagged configuration accepted")
	}

	if err.Error() != "Discovery job [es/0]: addUntagged is not yet implemented for type es" {
		t.Error(err)
	}
}

func TestConfLoadAddUntaggedLambda(t *testing.T) {
	config = conf{}
	configFile := "config_test_adduntagged_lambda.yml"
	err := config.load(&configFile)

	if err != nil {
		t.Error(err)
	}
}

func TestConfLoadAddUntaggedSearchTags(t *testing.T) {
	config = conf{}
	configFile := "config_test_adduntagged_lambda_search.yml"
	err := config.load(&configFile)

	if err == nil {
		t.Error("Invalid addUntagged configuration accepted")
	}

	if err.Error() != "Discovery job [lambda/0]: addUntagged cannot be used with searchTags (it would never match)" {
		t.Error(err)
	}
}
