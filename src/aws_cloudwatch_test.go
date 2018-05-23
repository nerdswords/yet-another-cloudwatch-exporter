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

func buildDatapoints(points []int, datapointsType string) []*cloudwatch.Datapoint {
	datapoints := []*cloudwatch.Datapoint{}

	for _, p := range points {
		number := float64(p)
		point := cloudwatch.Datapoint{}
		switch datapointsType {
		case "Minimum":
			point.Minimum = &number
		}

		datapoints = append(datapoints, &point)
	}

	return datapoints
}

var cloudwatchtests = []struct {
	datapoints     []int
	datapointsType string
	arn            string
	service        string
	statistics     string
	exported       string
	result         cloudwatchData
}{
	{[]int{}, "Maximum", "arn:aws:someservice:eu-west-1:xxxxxxxxxxxx:some-type/just-som-valid-arn", "ec2", "Minimum", "First", cloudwatchData{Id: aws.String("arn:aws:someservice:eu-west-1:xxxxxxxxxxxx:some-type/just-som-valid-arn")}},
}

func TestGetCloudwatchData(t *testing.T) {
	for _, tt := range cloudwatchtests {

		resp := cloudwatch.GetMetricStatisticsOutput{
			Datapoints: buildDatapoints(tt.datapoints, tt.datapointsType),
		}

		mock := cloudwatchInterface{
			client: mockedReceiveMsgs{Resp: resp},
		}

		resource := awsInfoData{
			Id:      aws.String(tt.arn),
			Service: aws.String(tt.service),
		}

		metric := metric{
			Statistics: tt.statistics,
			Exported:   tt.exported,
		}

		output := *mock.get(&resource, metric)

		verify := output

		if *verify.Id != *tt.result.Id {
			t.Errorf("handler returned unexpected body: got %v want %v",
				*verify.Id, *tt.result.Id)
		}
	}
}
