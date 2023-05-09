package v2

import (
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewClientCache(t *testing.T) {
	role1 := config.Role{
		RoleArn:    "role1",
		ExternalID: "external1",
	}
	role2 := config.Role{
		RoleArn:    "role2",
		ExternalID: "external2",
	}
	role3 := config.Role{
		RoleArn:    "role3",
		ExternalID: "external3",
	}

	region1 := "region1"
	region2 := "region2"
	region3 := "region3"
	tests := []struct {
		name       string
		config     config.ScrapeConf
		fips       bool
		onlyStatic bool
	}{
		{
			name: "initializes clients from discovery config",
			config: config.ScrapeConf{
				Discovery: config.Discovery{
					ExportedTagsOnMetrics: nil,
					Jobs: []*config.Job{
						{
							Regions: []string{region1, region2, region3},
							Roles:   []config.Role{role1, role2, role3},
						},
					},
				},
			},
			fips:       false,
			onlyStatic: false,
		}, {
			name: "initializes clients from static config",
			config: config.ScrapeConf{
				Static: []*config.Static{{
					Regions: []string{region1, region2, region3},
					Roles:   []config.Role{role1, role2, role3},
				}},
			},
			fips:       false,
			onlyStatic: true,
		}, {
			name: "initializes clients from custom config",
			config: config.ScrapeConf{
				CustomNamespace: []*config.CustomNamespace{{
					Regions: []string{region1, region2, region3},
					Roles:   []config.Role{role1, role2, role3},
				}},
			},
			fips:       false,
			onlyStatic: false,
		},
		{
			name: "initializes clients from all configs",
			config: config.ScrapeConf{
				Discovery: config.Discovery{
					ExportedTagsOnMetrics: nil,
					Jobs: []*config.Job{
						{
							Regions: []string{region1, region2},
							Roles:   []config.Role{role1, role2, role3},
						},
					},
				},
				Static: []*config.Static{{
					Regions: []string{region2, region3},
					Roles:   []config.Role{role2, role3},
				}},
				CustomNamespace: []*config.CustomNamespace{{
					Regions: []string{region1, region3},
					Roles:   []config.Role{role1, role2, role3},
				}},
			},
			fips:       false,
			onlyStatic: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, err := NewCache(test.config, test.fips, logging.NewNopLogger())
			require.NoError(t, err)
			cache := output.(*clientCache)

			assert.False(t, cache.refreshed)
			assert.False(t, cache.cleared)

			assert.Len(t, cache.clients, 3)
			assert.Contains(t, cache.clients, role1)
			assert.Contains(t, cache.clients, role2)
			assert.Contains(t, cache.clients, role3)

			for role, regionalClients := range cache.clients {
				assert.Len(t, regionalClients, 3)

				assert.Contains(t, regionalClients, region1)
				assert.Contains(t, regionalClients, region2)
				assert.Contains(t, regionalClients, region3)

				for region, clients := range regionalClients {
					assert.NotNil(t, clients, "role %s region %s had nil clients", role, region)
					assert.Equal(t, test.onlyStatic, clients.onlyStatic, "role %s region %s had unexpected onlyStatic value", role, region)
				}
			}
		})
	}
}
