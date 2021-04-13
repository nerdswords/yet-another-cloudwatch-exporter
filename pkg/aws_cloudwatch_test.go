package exporter

import (
	"testing"
	"time"

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

// TestSortyByTimeStamp validates that sortByTimestamp() sorts in descending order.
func TestSortyByTimeStamp(t *testing.T) {
	cloudWatchDataPoints := make([]*cloudwatch.Datapoint, 3)
	maxValue1 := float64(1)
	maxValue2 := float64(2)
	maxValue3 := float64(3)

	dataPointMiddle := &cloudwatch.Datapoint{}
	twoMinutesAgo := time.Now().Add(time.Minute * 2 * -1)
	dataPointMiddle.Timestamp = &twoMinutesAgo
	dataPointMiddle.Maximum = &maxValue2
	cloudWatchDataPoints[0] = dataPointMiddle

	dataPointNewest := &cloudwatch.Datapoint{}
	oneMinutesAgo := time.Now().Add(time.Minute * -1)
	dataPointNewest.Timestamp = &oneMinutesAgo
	dataPointNewest.Maximum = &maxValue1
	cloudWatchDataPoints[1] = dataPointNewest

	dataPointOldest := &cloudwatch.Datapoint{}
	threeMinutesAgo := time.Now().Add(time.Minute * 3 * -1)
	dataPointOldest.Timestamp = &threeMinutesAgo
	dataPointOldest.Maximum = &maxValue3
	cloudWatchDataPoints[2] = dataPointOldest

	sortedDataPoints := sortByTimestamp(cloudWatchDataPoints)

	equals(t, maxValue1, *sortedDataPoints[0].Maximum)
	equals(t, maxValue2, *sortedDataPoints[1].Maximum)
	equals(t, maxValue3, *sortedDataPoints[2].Maximum)
}
