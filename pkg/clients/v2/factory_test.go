package v2

import (
	"context"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	aws_config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cloudwatch_client "github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/clients/cloudwatch"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

func TestNewClientCache_initializes_clients(t *testing.T) {
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
		onlyStatic *bool
	}{
		{
			name: "from discovery config",
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
			onlyStatic: aws.Bool(false),
		},
		{
			name: "from static config",
			config: config.ScrapeConf{
				Static: []*config.Static{{
					Regions: []string{region1, region2, region3},
					Roles:   []config.Role{role1, role2, role3},
				}},
			},
			onlyStatic: aws.Bool(true),
		},
		{
			name: "from custom config",
			config: config.ScrapeConf{
				CustomNamespace: []*config.CustomNamespace{{
					Regions: []string{region1, region2, region3},
					Roles:   []config.Role{role1, role2, role3},
				}},
			},
			onlyStatic: aws.Bool(true),
		},
		{
			name: "from all configs",
			config: config.ScrapeConf{
				Discovery: config.Discovery{
					ExportedTagsOnMetrics: nil,
					Jobs: []*config.Job{
						{
							Regions: []string{region1, region2},
							Roles:   []config.Role{role1, role2},
						},
					},
				},
				Static: []*config.Static{{
					Regions: []string{region2, region3},
					Roles:   []config.Role{role2, role3},
				}},
				CustomNamespace: []*config.CustomNamespace{{
					Regions: []string{region1, region3},
					Roles:   []config.Role{role1, role3},
				}},
			},
			onlyStatic: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, err := NewFactory(test.config, false, logging.NewNopLogger())
			require.NoError(t, err)

			assert.False(t, output.refreshed)
			assert.False(t, output.cleared)

			assert.Len(t, output.clients, 3)
			assert.Contains(t, output.clients, role1)
			assert.Contains(t, output.clients, role2)
			assert.Contains(t, output.clients, role3)

			for role, regionalClients := range output.clients {
				assert.Len(t, regionalClients, 3)

				assert.Contains(t, regionalClients, region1)
				assert.Contains(t, regionalClients, region2)
				assert.Contains(t, regionalClients, region3)

				for region, clients := range regionalClients {
					assert.NotNil(t, clients, "role %s region %s had nil clients", role, region)
					if test.onlyStatic != nil {
						assert.Equal(t, *test.onlyStatic, clients.onlyStatic, "role %s region %s had unexpected onlyStatic value", role, region)
					}
				}
			}
		})
	}
}

func TestNewClientCache_sets_fips(t *testing.T) {
	config := config.ScrapeConf{
		Discovery: config.Discovery{
			ExportedTagsOnMetrics: nil,
			Jobs: []*config.Job{
				{
					Roles:   []config.Role{{}},
					Regions: []string{"region1"},
				},
			},
		},
	}
	output, err := NewFactory(config, true, logging.NewNopLogger())
	require.NoError(t, err)

	clients := output.clients[defaultRole]["region1"]
	assert.NotNil(t, clients)

	foundLoadOptions := false
	for _, sources := range clients.awsConfig.ConfigSources {
		options, ok := sources.(aws_config.LoadOptions)
		if !ok {
			continue
		}
		foundLoadOptions = true
		assert.Equal(t, aws.FIPSEndpointStateEnabled, options.UseFIPSEndpoint)
	}
	assert.True(t, foundLoadOptions)
}

func TestNewClientCache_sets_endpoint_override(t *testing.T) {
	config := config.ScrapeConf{
		Discovery: config.Discovery{
			ExportedTagsOnMetrics: nil,
			Jobs: []*config.Job{
				{
					Roles:   []config.Role{{}},
					Regions: []string{"region1"},
				},
			},
		},
	}

	err := os.Setenv("AWS_ENDPOINT_URL", "https://totallynotaws.com")
	require.NoError(t, err)

	output, err := NewFactory(config, false, logging.NewNopLogger())
	require.NoError(t, err)

	clients := output.clients[defaultRole]["region1"]
	assert.NotNil(t, clients)
	assert.NotNil(t, clients.awsConfig.EndpointResolverWithOptions)
}

func TestClientCache_Clear(t *testing.T) {
	cache := &CachingFactory{
		logger: logging.NewNopLogger(),
		clients: map[config.Role]map[awsRegion]*cachedClients{
			defaultRole: {
				"region1": &cachedClients{
					awsConfig:  nil,
					cloudwatch: testClient{},
					tagging:    testClient{},
					account:    testClient{},
				},
			},
		},
		refreshed: true,
		cleared:   false,
	}

	cache.Clear()
	assert.True(t, cache.cleared)
	assert.False(t, cache.refreshed)

	clients := cache.clients[defaultRole]["region1"]
	require.NotNil(t, clients)
	assert.Nil(t, clients.cloudwatch)
	assert.Nil(t, clients.account)
	assert.Nil(t, clients.tagging)
}

func TestClientCache_Refresh(t *testing.T) {
	t.Run("creates all clients when config contains only discovery jobs", func(t *testing.T) {
		config := config.ScrapeConf{
			Discovery: config.Discovery{
				ExportedTagsOnMetrics: nil,
				Jobs: []*config.Job{
					{
						Roles:   []config.Role{{}},
						Regions: []string{"region1"},
					},
				},
			},
		}

		output, err := NewFactory(config, false, logging.NewNopLogger())
		require.NoError(t, err)

		output.Refresh()
		assert.False(t, output.cleared)
		assert.True(t, output.refreshed)

		clients := output.clients[defaultRole]["region1"]
		require.NotNil(t, clients)
		assert.NotNil(t, clients.cloudwatch)
		assert.NotNil(t, clients.account)
		assert.NotNil(t, clients.tagging)
	})

	t.Run("creates only cloudwatch when config is only static jobs", func(t *testing.T) {
		config := config.ScrapeConf{
			Static: []*config.Static{{
				Regions: []string{"region1"},
				Roles:   []config.Role{{}},
			}},
			CustomNamespace: []*config.CustomNamespace{{
				Regions: []string{"region1"},
				Roles:   []config.Role{{}},
			}},
		}

		output, err := NewFactory(config, false, logging.NewNopLogger())
		require.NoError(t, err)

		output.Refresh()
		assert.False(t, output.cleared)
		assert.True(t, output.refreshed)

		clients := output.clients[defaultRole]["region1"]
		require.NotNil(t, clients)
		assert.NotNil(t, clients.cloudwatch)
		assert.Nil(t, clients.account)
		assert.Nil(t, clients.tagging)
	})
}

func TestClientCache_GetAccountClient(t *testing.T) {
	t.Run("refreshed cache does not create new client", func(t *testing.T) {
		config := config.ScrapeConf{
			Discovery: config.Discovery{
				ExportedTagsOnMetrics: nil,
				Jobs: []*config.Job{
					{
						Roles:   []config.Role{{}},
						Regions: []string{"region1"},
					},
				},
			},
		}

		output, err := NewFactory(config, false, logging.NewNopLogger())
		require.NoError(t, err)

		output.Refresh()

		clients := output.clients[defaultRole]["region1"]
		require.NotNil(t, clients)
		assert.Equal(t, clients.account, output.GetAccountClient("region1", defaultRole))
	})

	t.Run("unrefreshed cache creates a new client", func(t *testing.T) {
		config := config.ScrapeConf{
			Discovery: config.Discovery{
				ExportedTagsOnMetrics: nil,
				Jobs: []*config.Job{
					{
						Roles:   []config.Role{{}},
						Regions: []string{"region1"},
					},
				},
			},
		}

		output, err := NewFactory(config, false, logging.NewNopLogger())
		require.NoError(t, err)

		clients := output.clients[defaultRole]["region1"]
		require.NotNil(t, clients)
		require.Nil(t, clients.account)

		client := output.GetAccountClient("region1", defaultRole)
		assert.Equal(t, clients.account, client)
	})
}

func TestClientCache_GetCloudwatchClient(t *testing.T) {
	t.Run("refreshed cache does not create new client", func(t *testing.T) {
		config := config.ScrapeConf{
			Discovery: config.Discovery{
				ExportedTagsOnMetrics: nil,
				Jobs: []*config.Job{
					{
						Roles:   []config.Role{{}},
						Regions: []string{"region1"},
					},
				},
			},
		}

		output, err := NewFactory(config, false, logging.NewNopLogger())
		require.NoError(t, err)

		output.Refresh()

		clients := output.clients[defaultRole]["region1"]
		require.NotNil(t, clients)
		// Can't do equality comparison due to concurrency limiter
		assert.NotNil(t, output.GetCloudwatchClient("region1", defaultRole, 1))
	})

	t.Run("unrefreshed cache creates a new client", func(t *testing.T) {
		config := config.ScrapeConf{
			Discovery: config.Discovery{
				ExportedTagsOnMetrics: nil,
				Jobs: []*config.Job{
					{
						Roles:   []config.Role{{}},
						Regions: []string{"region1"},
					},
				},
			},
		}

		output, err := NewFactory(config, false, logging.NewNopLogger())
		require.NoError(t, err)

		clients := output.clients[defaultRole]["region1"]
		require.NotNil(t, clients)
		require.Nil(t, clients.cloudwatch)

		output.GetCloudwatchClient("region1", defaultRole, 1)
		assert.NotNil(t, clients.cloudwatch)
	})
}

func TestClientCache_GetTaggingClient(t *testing.T) {
	t.Run("refreshed cache does not create new client", func(t *testing.T) {
		config := config.ScrapeConf{
			Discovery: config.Discovery{
				ExportedTagsOnMetrics: nil,
				Jobs: []*config.Job{
					{
						Roles:   []config.Role{{}},
						Regions: []string{"region1"},
					},
				},
			},
		}

		output, err := NewFactory(config, false, logging.NewNopLogger())
		require.NoError(t, err)

		output.Refresh()

		clients := output.clients[defaultRole]["region1"]
		require.NotNil(t, clients)
		// Can't do equality comparison due to concurrency limiter
		assert.NotNil(t, output.GetTaggingClient("region1", defaultRole, 1))
	})

	t.Run("unrefreshed cache creates a new client", func(t *testing.T) {
		config := config.ScrapeConf{
			Discovery: config.Discovery{
				ExportedTagsOnMetrics: nil,
				Jobs: []*config.Job{
					{
						Roles:   []config.Role{{}},
						Regions: []string{"region1"},
					},
				},
			},
		}

		output, err := NewFactory(config, false, logging.NewNopLogger())
		require.NoError(t, err)

		clients := output.clients[defaultRole]["region1"]
		require.NotNil(t, clients)
		require.Nil(t, clients.tagging)

		output.GetTaggingClient("region1", defaultRole, 1)
		assert.NotNil(t, clients.tagging)
	})
}

type testClient struct{}

func (t testClient) GetResources(_ context.Context, _ *config.Job, _ string) ([]*model.TaggedResource, error) {
	return nil, nil
}

func (t testClient) GetAccount(_ context.Context) (string, error) {
	return "", nil
}

func (t testClient) ListMetrics(_ context.Context, _ string, _ *config.Metric, _ bool, _ func(page []*model.Metric)) ([]*model.Metric, error) {
	return nil, nil
}

func (t testClient) GetMetricData(_ context.Context, _ logging.Logger, _ []*model.CloudwatchData, _ string, _ int64, _ int64, _ *int64) []*cloudwatch_client.MetricDataResult {
	return nil
}

func (t testClient) GetMetricStatistics(_ context.Context, _ logging.Logger, _ []*model.Dimension, _ string, _ *config.Metric) []*model.Datapoint {
	return nil
}
