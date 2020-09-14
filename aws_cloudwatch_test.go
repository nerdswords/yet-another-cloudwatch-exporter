package main

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

func TestDimensionsToCliString(t *testing.T) {
	// Setup Test

	// Arrange
	dimensions := []*cloudwatch.Dimension{}
	expected := ""

	// Act
	actual := dimensionsToCliString(dimensions)

	// Assert
	if actual != expected {
		t.Fatalf("\nexpected: %q\nactual:  %q", expected, actual)
	}
}

func TestGetNamespace(t *testing.T) {
	for _, jobType := range supportedServices {
		ns, err := getNamespace(jobType)
		if err != nil {
			t.Fatalf("jobType %s shouldn't have returned error", jobType)
		}
		if ns == "" {
			t.Fatalf("jobType %s seems to have empty namespace", jobType)
		}
	}
	ns, err := getNamespace("foobar")
	if !strings.Contains(err.Error(), "foobar") {
		t.Fatalf("jobType foobar should have returned error")
	}
	if ns != "" {
		t.Fatalf("jobType foobar should have returned empty string")
	}
}
