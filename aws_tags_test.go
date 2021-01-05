package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"testing"
)

func TestMigrateTagsToPrometheus(t *testing.T) {
	// Setup Test
	id := "tag_Id"
	service := "tag_Service"
	region := "us-east-1"
	tagItem := tag{Key: "Name", Value: "tag_Value"}
	tags := []*tag{&tagItem}
	tagData := tagsData{ID: &id, Service: &service, Region: &region, Tags: tags}
	tagsData := []*tagsData{&tagData}

	// Arrange
	prometheusMetricName := "aws_tag_service_info"
	promLabels := make(map[string]string)
	promLabels["name"] = "tag_Id"
	promLabels["tag_Name"] = "tag_Value"
	var metricValue float64 = 0

	p := PrometheusMetric{
		name:   &prometheusMetricName,
		labels: promLabels,
		value:  &metricValue,
	}
	expected := []*PrometheusMetric{&p}

	// Act
	actual := migrateTagsToPrometheus(tagsData)

	// Assert
	if *actual[0].name != *expected[0].name {
		t.Fatalf("\nexpected: %q\nactual:  %q", len(expected), len(actual))
	}

}

func TestChunkArrayOfStrings(t *testing.T) {
	// Setup Test
	targetGroupArns := []*string{aws.String("a"),
		aws.String("b"),
		aws.String("c"),
		aws.String("d"),
		aws.String("e"),
		aws.String("f"),
		aws.String("g"),
		aws.String("h"),
	}

	expected := [][]*string{[]*string{targetGroupArns[0],
		targetGroupArns[1],
		targetGroupArns[2]},
		[]*string{
			targetGroupArns[3],
			targetGroupArns[4],
			targetGroupArns[5]},
		[]*string{
			targetGroupArns[6],
			targetGroupArns[7],
		},
	}

	// Act
	actual := chunkArrayOfStrings(targetGroupArns, 3)

	actualString := fmt.Sprintf("%#v\n", actual)
	expectedString := fmt.Sprintf("%#v\n", expected)

	// Assert
	if len(actualString) != len(expectedString) {
		t.Fatalf("\nexpected: %q\nactual:  %q", len(expected), len(actual))
	}
	if actualString != expectedString {
		t.Fatalf("\nexpected: %q\nactual:  %q", expectedString, actualString)
	}
}
