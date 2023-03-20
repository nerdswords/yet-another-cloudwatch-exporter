package config

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/require"
)

func TestSupportedServices(t *testing.T) {
	for i, svc := range SupportedServices {
		require.NotNil(t, svc.Namespace, fmt.Sprintf("Nil Namespace for service at index '%d'", i))
		require.NotNil(t, svc.Alias, fmt.Sprintf("Nil Alias for service '%s' at index '%d'", svc.Namespace, i))

		if svc.ResourceFilters != nil {
			require.NotEmpty(t, svc.ResourceFilters)

			for _, filter := range svc.ResourceFilters {
				require.NotEmpty(t, aws.StringValue(filter))
			}
		}

		if svc.DimensionRegexps != nil {
			require.NotEmpty(t, svc.DimensionRegexps)

			for _, regex := range svc.DimensionRegexps {
				require.NotEmpty(t, regex.String())
				require.Positive(t, regex.NumSubexp())
				names := regex.SubexpNames()
				// Avoid looking at the first capturing group, which captures the whole expression
				for _, name := range names[1:] {
					require.NotEmpty(t, name, "%s DimensionRegexps shouldn't use not-named capturing groups", svc.Namespace)
				}
			}
		}
	}
}

// TestDimensionRegexps tests that the added DimensionRegexps matches the expected dimensions, given an ARN. These serves
// as a first test layer, since the matches are later used in the resource matching algorithm.
func TestDimensionRegexps(t *testing.T) {
	type args struct {
		serviceType string
		arn         string
	}
	type testCase struct {
		args     args
		expected map[string]string
	}

	for name, tc := range map[string]testCase{
		"AWS/ECS service resource": {
			args: args{
				serviceType: "AWS/ECS",
				arn:         "arn:aws:ecs:us-east-1:123:service/scorekeep-cluster/scorekeep-service",
			},
			expected: map[string]string{
				"ServiceName": "scorekeep-service",
				"ClusterName": "scorekeep-cluster",
			},
		},
		"AWS/ECS cluster resource": {
			args: args{
				serviceType: "AWS/ECS",
				arn:         "arn:aws:ecs:us-east-1:123:cluster/scorekeep-cluster",
			},
			expected: map[string]string{
				"ClusterName": "scorekeep-cluster",
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			regexps := SupportedServices.GetService(tc.args.serviceType).DimensionRegexps

			matchResults := map[string]string{}

			for _, regex := range regexps {
				names := regex.SubexpNames()
				for i, res := range regex.FindStringSubmatch(tc.args.arn) {
					if i == 0 {
						continue
					}
					matchResults[names[i]] = res
				}
			}

			require.Equal(t, tc.expected, matchResults)
		})
	}
}
