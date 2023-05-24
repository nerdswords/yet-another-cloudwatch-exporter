package v1

import (
	"testing"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

func TestDimensionsToCliString(t *testing.T) {
	// Setup Test

	// Arrange
	dimensions := []*model.Dimension{}
	expected := ""

	// Act
	actual := dimensionsToCliString(dimensions)

	// Assert
	if actual != expected {
		t.Fatalf("\nexpected: %q\nactual:  %q", expected, actual)
	}
}
