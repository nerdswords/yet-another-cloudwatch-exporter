package exporter

import (
	"testing"
)

func TestFilterThroughTags(t *testing.T) {
	// Setup Test

	// Arrange
	expected := true
	tagsData := taggedResource{}
	filterTags := []Tag{}

	// Act
	actual := tagsData.filterThroughTags(filterTags)

	// Assert
	if actual != expected {
		t.Fatalf("\nexpected: %t\nactual:  %t", expected, actual)
	}
}
