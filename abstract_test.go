package main

import (
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"reflect"
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

// set up mock cloudwatch client for TestResolveSingleStaticDimension
type mockCloudWatchClient struct {
	cloudwatchiface.CloudWatchAPI
}

func (m mockCloudWatchClient) ListMetrics(input *cloudwatch.ListMetricsInput) (*cloudwatch.ListMetricsOutput, error) {
	fixedDimensionName := "fixedDimension"
	fixedDimensionValue := "fixedValue"
	variableDimensionName := "variableDimensionName"
	variableDimensionValue1 := "a"
	variableDimensionValue2 := "b"
	fixedDimension := cloudwatch.Dimension{
		Name:  &fixedDimensionName,
		Value: &fixedDimensionValue,
	}
	variableDimension1 := cloudwatch.Dimension{
		Name:  &variableDimensionName,
		Value: &variableDimensionValue1,
	}
	variableDimension2 := cloudwatch.Dimension{
		Name:  &variableDimensionName,
		Value: &variableDimensionValue2,
	}
	dimensions1 := []*cloudwatch.Dimension{&fixedDimension, &variableDimension1}
	dimensions2 := []*cloudwatch.Dimension{&fixedDimension, &variableDimension2}
	metric1 := cloudwatch.Metric{
		Dimensions: dimensions1,
		MetricName: input.MetricName,
		Namespace:  input.Namespace,
	}
	metric2 := cloudwatch.Metric{
		Dimensions: dimensions2,
		MetricName: input.MetricName,
		Namespace:  input.Namespace,
	}
	metrics := []*cloudwatch.Metric{&metric1, &metric2}
	output := cloudwatch.ListMetricsOutput{
		Metrics:   metrics,
		NextToken: nil,
	}
	return &output, nil
}

func TestMapToOrderedStringHelper(t *testing.T) {
	// Setup Test
	m1 := map[string]string{"a": "1", "b": "2"}
	m2 := map[string]string{"b": "2", "a": "1"}
	m3 := map[string]string{"a": "0", "b": "2"}
	m4 := map[string]string{"a": "1", "b": "2", "c": "3"}

	// Act
	s1 := mapToOrderedStringHelper(m1)
	s2 := mapToOrderedStringHelper(m2)
	s3 := mapToOrderedStringHelper(m3)
	s4 := mapToOrderedStringHelper(m4)

	// Assert
	if !(s1 == s2) {
		t.Fatalf("expected %s == %s", s1, s2)
	}
	if s1 == s3 {
		t.Fatalf("expected %s != %s", s1, s3)
	}
	if s1 == s4 {
		t.Fatalf("expected %s != %s", s1, s4)
	}
}

func TestResolveSingleStaticDimension(t *testing.T) {
	// Setup Test
	dimensions := []dimension{
		{
			Name:  "fixedDimension",
			Value: "fixedValue",
		},
		{
			Name: "variableDimensionName", // variable dimension takes values "a" and "b"
		},
	}
	metrics := []metric{
		{
			Name:       "metric1",
			Statistics: []string{"average"},
		},
		{
			Name:       "metric2",
			Statistics: []string{"average"},
		},
	}
	resource := static{
		Name:                       "simpleStaticTest",
		Regions:                    []string{"us-east-1"},
		RoleArns:                   []string{"roleARN"},
		Namespace:                  "myCustomNamespace",
		Dimensions:                 dimensions,
		Metrics:                    metrics,
		PopulateNamelessDimensions: true,
	}
	clientCloudwatch := cloudwatchInterface{
		client: mockCloudWatchClient{},
	}

	var expectedResults []static
	expectedVariableDimensions := []dimension{
		{
			Name:  "variableDimensionName",
			Value: "a",
		},
		{
			Name:  "variableDimensionName",
			Value: "b",
		},
	}
	for _, value := range expectedVariableDimensions {
		result := resource
		resultDimensions := []dimension{result.Dimensions[0], value}
		result.Dimensions = resultDimensions
		expectedResults = append(expectedResults, result)
	}
	// Act
	newJobs, err := resolveStaticDimensions(resource, clientCloudwatch)

	// Assert
	if err != nil {
		t.Fatalf("Unexpected error from resolveStaticDimensions, %e", err)
	}
	if len(newJobs) != 2 {
		t.Fatalf("Expected two new jobs to be created, got %d", len(newJobs))
	}
	newJobStructs := []static{*newJobs[0], *newJobs[1]}
	if !reflect.DeepEqual(newJobStructs, expectedResults) {
		t.Fatalf("\nexpected: %+v\nactual:  %+v", expectedResults, newJobStructs)
	}
}
