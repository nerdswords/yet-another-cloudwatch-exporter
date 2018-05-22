package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"testing"
)

type mockedReceiveMsgs struct {
	cloudwatchiface.CloudWatchAPI
	Resp cloudwatch.GetMetricStatisticsOutput
}

func (m mockedReceiveMsgs) GetMetricStatistics(in *cloudwatch.GetMetricStatisticsInput) (*cloudwatch.GetMetricStatisticsOutput, error) {
	return &m.Resp, nil
}

func buildDatapoints(points []int) []*cloudwatch.Datapoint {
	datapoints := []*cloudwatch.Datapoint{}

	for _, p := range points {
		number := float64(p)
		point := cloudwatch.Datapoint{
			Minimum: &number,
		}
		datapoints = append(datapoints, &point)
	}

	return datapoints
}

func TestGetCloudwatchData(t *testing.T) {
	resp := cloudwatch.GetMetricStatisticsOutput{
		Datapoints: buildDatapoints([]int{6, 4, 1, 5}),
	}

	mock := cloudwatchInterface{
		client: mockedReceiveMsgs{Resp: resp},
	}

	resource := awsInfoData{
		Id:      aws.String("arn:aws:elasticache:eu-west-1:xxxxxxxxxxxx:instance/i-01ccc2af5414c54a7"),
		Service: aws.String("ec2"),
	}

	metric := metric{
		Statistics: "Minimum",
		Exported:   "First",
	}

	output := *mock.get(&resource, metric)

	verify := *output.Value
	expected := float64(6)

	if verify != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			verify, expected)
	}
}
