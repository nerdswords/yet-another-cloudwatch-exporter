package job

import (
	"testing"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/services"
)

func TestFilterThroughTags(t *testing.T) {
	// Setup Test

	// Arrange
	expected := true
	tagsData := services.TaggedResource{}
	filterTags := []model.Tag{}

	// Act
	actual := tagsData.FilterThroughTags(filterTags)

	// Assert
	if actual != expected {
		t.Fatalf("\nexpected: %t\nactual:  %t", expected, actual)
	}
}
