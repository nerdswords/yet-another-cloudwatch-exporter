package tagging_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/tagging/v1"
	v2 "github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/tagging/v2"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
)

func Test_Services_Have_Filters_In_V1_and_V2(t *testing.T) {
	for _, service := range config.SupportedServices {
		namespace := service.Namespace
		t.Run(fmt.Sprintf("%s has filter definitions in v1 and v2", namespace), func(t *testing.T) {
			v1Filters, v1Exists := v1.ServiceFilters[namespace]
			v2Filters, v2Exists := v2.ServiceFilters[namespace]

			require.Equal(t, v1Exists, v2Exists, "Service filters are only implemented for v1 or v2 but should be implemented for both")

			v1FilterFuncNil := v1Filters.FilterFunc == nil
			v2FilterFuncNil := v2Filters.FilterFunc == nil
			assert.Equal(t, v1FilterFuncNil, v2FilterFuncNil, "FilterFunc is only implemented for v1 or v2 but should be implemented for both")

			v1ResourceFuncNil := v1Filters.ResourceFunc == nil
			v2ResourceFuncNil := v2Filters.ResourceFunc == nil
			assert.Equal(t, v1ResourceFuncNil, v2ResourceFuncNil, "ResourceFunc is only implemented for v1 or v2 but should be implemented for both")
		})
	}
}
