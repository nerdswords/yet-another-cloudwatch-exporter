package job

import (
	"testing"

	"github.com/brianvoe/gofakeit/v6"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

func BenchmarkSlices(b *testing.B) {
	type testcase struct {
		jobs          int
		resultsPerJob int
	}

	for name, tc := range map[string]testcase{
		"1 job large results": {
			jobs:          1,
			resultsPerJob: 500000,
		},
		"1 job medium results": {
			jobs:          1,
			resultsPerJob: 100000,
		},
		"1 job small results": {
			jobs:          1,
			resultsPerJob: 1000,
		},
		"multiple jobs large results": {
			jobs:          5,
			resultsPerJob: 100000,
		},
		"multiple jobs medium results": {
			jobs:          5,
			resultsPerJob: 10000,
		},
		"multiple jobs small results": {
			jobs:          5,
			resultsPerJob: 100,
		},
	} {
		input := setupInputData(tc.jobs, tc.resultsPerJob)

		b.Run(name+" slice of slices", func(b *testing.B) {
			doSliceOfSlicesBench(b, input)
		})

		b.Run(name+" slice concat", func(b *testing.B) {
			doSliceConcatBench(b, input)
		})
	}
}

func setupInputData(jobs int, resultsPerJob int) [][]*model.CloudwatchData {
	output := make([][]*model.CloudwatchData, jobs, jobs)
	var data model.CloudwatchData
	err := gofakeit.Struct(&data)
	if err != nil {
		panic(err)
	}
	for i := 0; i < jobs; i++ {
		output[i] = make([]*model.CloudwatchData, resultsPerJob, resultsPerJob)
		for j := 0; j < resultsPerJob; j++ {
			output[i][j] = &data
		}
	}
	return output
}

func doSliceOfSlicesBench(b *testing.B, resultData [][]*model.CloudwatchData) {
	for i := 0; i < b.N; i++ {
		results := make([][]*model.CloudwatchData, 0)
		for _, result := range resultData {
			results = append(results, result)
		}
	}
}

func doSliceConcatBench(b *testing.B, resultData [][]*model.CloudwatchData) {
	for i := 0; i < b.N; i++ {
		results := make([]*model.CloudwatchData, 0)
		for _, result := range resultData {
			results = append(results, result...)
		}
	}
}
