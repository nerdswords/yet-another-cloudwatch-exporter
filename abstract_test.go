package main

import (
	"testing"
)

func TestFilterThroughTags(t *testing.T) {
	// Setup Test

	// Arrange
	expected := true
	tagsData := tagsData{}
	filterTags := []tag{}

	// Act
	actual := tagsData.filterThroughTags(filterTags)

	// Assert
	if actual != expected {
		t.Fatalf("\nexpected: %t\nactual:  %t", expected, actual)
	}
}
