package v1

import (
	"fmt"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/awstesting/mock"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/config"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
)

func cmpCache(t *testing.T, initialCache *clientCache, cache *clientCache) {
	for role := range initialCache.stscache {
		if _, ok := cache.stscache[role]; !ok {
			t.Logf("`role` not in sts cache %s", role.RoleArn)
			t.Fail()
		}
	}

	for role, regionMap := range initialCache.clients {
		if _, ok := cache.clients[role]; !ok {
			t.Logf("`role` not in client cache %s", role.RoleArn)
			t.Fail()
			continue
		}

		for region, client := range regionMap {
			if _, ok := cache.clients[role][region]; !ok {
				t.Logf("`region` %s not found in role %s", region, role.RoleArn)
				t.Fail()
			}

			if client == nil {
				t.Logf("`client cache` is nil for region %s and role %v", region, role)
				continue
			}

			if cache.clients[role][region] == nil {
				t.Logf("comparison `client cache` is nil for region %s and role %v", region, role)
				continue
			}

			if *client != *cache.clients[role][region] {
				t.Logf("`client` %v is not equal to %v for role %v in region %s", *client, *cache.clients[role][region], role, region)
				t.Logf("The cache for this client is %v", cache.clients[role])
				t.Logf("The cache for the comparison client is %v", client)
				t.Fail()
			}
		}
	}
}

func TestNewClientCache(t *testing.T) {
	tests := []struct {
		descrip string
		config  config.ScrapeConf
		fips    bool
		cache   *clientCache
	}{
		{
			"an empty config gives an empty cache",
			config.ScrapeConf{},
			false,
			&clientCache{logger: logging.NewNopLogger()},
		},
		{
			"if fips is set then the clients has fips",
			config.ScrapeConf{},
			true,
			&clientCache{
				fips:   true,
				logger: logging.NewNopLogger(),
			},
		},
		{
			"a ScrapeConf with only discovery jobs creates a cache",
			config.ScrapeConf{
				Discovery: config.Discovery{
					Jobs: []*config.Job{
						{
							Regions: []string{"us-east-1", "us-west-2", "ap-northeast-3"},
							Roles: []config.Role{
								{
									RoleArn: "some-arn",
								},
								{
									RoleArn: "some-arn2",
								},
								{
									RoleArn:    "some-arn",
									ExternalID: "thing",
								},
							},
						},
						{
							Regions: []string{"ap-northeast-3"},
							Roles: []config.Role{
								{
									RoleArn: "some-arn",
								},
								{
									RoleArn: "some-arn5",
								},
							},
						},
					},
				},
			},
			false,
			&clientCache{
				stscache: map[config.Role]stsiface.STSAPI{
					{RoleArn: "some-arn"}:                      nil,
					{RoleArn: "some-arn", ExternalID: "thing"}: nil,
					{RoleArn: "some-arn2"}:                     nil,
					{RoleArn: "some-arn5"}:                     nil,
				},
				clients: map[config.Role]map[string]*cachedClients{
					{RoleArn: "some-arn"}: {
						"ap-northeast-3": &cachedClients{},
						"us-east-1":      &cachedClients{},
						"us-west-2":      &cachedClients{},
					},
					{RoleArn: "some-arn", ExternalID: "thing"}: {
						"ap-northeast-3": &cachedClients{},
						"us-east-1":      &cachedClients{},
						"us-west-2":      &cachedClients{},
					},
					{RoleArn: "some-arn2"}: {
						"ap-northeast-3": &cachedClients{},
						"us-east-1":      &cachedClients{},
						"us-west-2":      &cachedClients{},
					},
					{RoleArn: "some-arn5"}: {
						"ap-northeast-3": &cachedClients{},
					},
				},
				logger: logging.NewNopLogger(),
			},
		},
		{
			"a ScrapeConf with only static jobs creates a cache",
			config.ScrapeConf{
				Static: []*config.Static{
					{
						Name:    "scrape-thing",
						Regions: []string{"us-east-1", "eu-west-2"},
						Roles: []config.Role{
							{
								RoleArn: "some-arn",
							},
							{
								RoleArn: "some-arn2",
							},
							{
								RoleArn: "some-arn3",
							},
						},
					},
					{
						Name:    "scrape-other-thing",
						Regions: []string{"us-east-1"},
						Roles: []config.Role{
							{
								RoleArn: "some-arn",
							},
							{
								RoleArn: "some-arn2",
							},
							{
								RoleArn:    "some-arn",
								ExternalID: "thing",
							},
						},
					},
					{
						Name:    "scrape-third-thing",
						Regions: []string{"ap-northeast-1"},
						Roles: []config.Role{
							{
								RoleArn: "some-arn",
							},
							{
								RoleArn: "some-arn2",
							},
							{
								RoleArn: "some-arn4",
							},
						},
					},
				},
			},
			false,
			&clientCache{
				stscache: map[config.Role]stsiface.STSAPI{
					{RoleArn: "some-arn"}:                      nil,
					{RoleArn: "some-arn", ExternalID: "thing"}: nil,
					{RoleArn: "some-arn2"}:                     nil,
					{RoleArn: "some-arn3"}:                     nil,
					{RoleArn: "some-arn4"}:                     nil,
				},
				clients: map[config.Role]map[string]*cachedClients{
					{RoleArn: "some-arn"}: {
						"ap-northeast-1": &cachedClients{onlyStatic: true},
						"eu-west-2":      &cachedClients{onlyStatic: true},
						"us-east-1":      &cachedClients{onlyStatic: true},
					},
					{RoleArn: "some-arn", ExternalID: "thing"}: {
						"us-east-1": &cachedClients{onlyStatic: true},
					},
					{RoleArn: "some-arn2"}: {
						"ap-northeast-1": &cachedClients{onlyStatic: true},
						"eu-west-2":      &cachedClients{onlyStatic: true},
						"us-east-1":      &cachedClients{onlyStatic: true},
					},
					{RoleArn: "some-arn3"}: {
						"eu-west-2": &cachedClients{onlyStatic: true},
						"us-east-1": &cachedClients{onlyStatic: true},
					},
					{RoleArn: "some-arn4"}: {
						"ap-northeast-1": &cachedClients{onlyStatic: true},
					},
				},
				logger: logging.NewNopLogger(),
			},
		},
		{
			"a ScrapeConf with some overlapping static and discovery jobs creates a cache",
			config.ScrapeConf{
				Discovery: config.Discovery{
					Jobs: []*config.Job{
						{
							Regions: []string{"us-east-1", "us-west-2", "ap-northeast-3"},
							Roles: []config.Role{
								{
									RoleArn: "some-arn",
								},
								{
									RoleArn: "some-arn2",
								},
								{
									RoleArn: "some-arn3",
								},
							},
						},
						{
							Regions: []string{"ap-northeast-3"},
							Roles: []config.Role{
								{
									RoleArn: "some-arn",
								},
								{
									RoleArn: "some-arn5",
								},
							},
						},
					},
				},
				Static: []*config.Static{
					{
						Name:    "scrape-thing",
						Regions: []string{"us-east-1", "eu-west-2"},
						Roles: []config.Role{
							{
								RoleArn: "some-arn",
							},
							{
								RoleArn: "some-arn2",
							},
							{
								RoleArn: "some-arn3",
							},
						},
					},
					{
						Name:    "scrape-other-thing",
						Regions: []string{"us-east-1"},
						Roles: []config.Role{
							{
								RoleArn: "some-arn",
							},
							{
								RoleArn: "some-arn2",
							},
							{
								RoleArn:    "some-arn",
								ExternalID: "thing",
							},
						},
					},
					{
						Name:    "scrape-third-thing",
						Regions: []string{"ap-northeast-1"},
						Roles: []config.Role{
							{
								RoleArn: "some-arn",
							},
							{
								RoleArn: "some-arn2",
							},
							{
								RoleArn: "some-arn4",
							},
						},
					},
				},
			},
			false,
			&clientCache{
				stscache: map[config.Role]stsiface.STSAPI{
					{RoleArn: "some-arn"}:                      nil,
					{RoleArn: "some-arn", ExternalID: "thing"}: nil,
					{RoleArn: "some-arn2"}:                     nil,
					{RoleArn: "some-arn3"}:                     nil,
					{RoleArn: "some-arn4"}:                     nil,
					{RoleArn: "some-arn5"}:                     nil,
				},
				clients: map[config.Role]map[string]*cachedClients{
					{RoleArn: "some-arn"}: {
						"ap-northeast-3": &cachedClients{},
						"us-east-1":      &cachedClients{},
						"ap-northeast-1": &cachedClients{onlyStatic: true},
						"eu-west-2":      &cachedClients{onlyStatic: true},
						"us-west-2":      &cachedClients{},
					},
					{RoleArn: "some-arn", ExternalID: "thing"}: {
						"us-east-1": &cachedClients{onlyStatic: true},
					},
					{RoleArn: "some-arn2"}: {
						"ap-northeast-3": &cachedClients{},
						"ap-northeast-1": &cachedClients{onlyStatic: true},
						"eu-west-2":      &cachedClients{onlyStatic: true},
						"us-east-1":      &cachedClients{},
						"us-west-2":      &cachedClients{},
					},
					{RoleArn: "some-arn3"}: {
						"ap-northeast-3": &cachedClients{},
						"eu-west-2":      &cachedClients{onlyStatic: true},
						"us-east-1":      &cachedClients{},
						"us-west-2":      &cachedClients{},
					},
					{RoleArn: "some-arn4"}: {
						"ap-northeast-1": &cachedClients{onlyStatic: true},
					},
					{RoleArn: "some-arn5"}: {
						"ap-northeast-3": &cachedClients{},
					},
				},
				logger: logging.NewNopLogger(),
			},
		},
		{
			"a ScrapeConf with only custom dimension jobs creates a cache",
			config.ScrapeConf{
				CustomNamespace: []*config.CustomNamespace{
					{
						Name:      "scrape-thing",
						Regions:   []string{"us-east-1", "eu-west-2"},
						Namespace: "CustomDimension",
						Roles: []config.Role{
							{
								RoleArn: "some-arn",
							},
							{
								RoleArn: "some-arn2",
							},
							{
								RoleArn: "some-arn3",
							},
						},
					},
					{
						Name:      "scrape-other-thing",
						Regions:   []string{"us-east-1"},
						Namespace: "CustomDimension",
						Roles: []config.Role{
							{
								RoleArn: "some-arn",
							},
							{
								RoleArn: "some-arn2",
							},
							{
								RoleArn:    "some-arn",
								ExternalID: "thing",
							},
						},
					},
					{
						Name:      "scrape-third-thing",
						Regions:   []string{"ap-northeast-1"},
						Namespace: "CustomDimension",
						Roles: []config.Role{
							{
								RoleArn: "some-arn",
							},
							{
								RoleArn: "some-arn2",
							},
							{
								RoleArn: "some-arn4",
							},
						},
					},
				},
			},
			false,
			&clientCache{
				stscache: map[config.Role]stsiface.STSAPI{
					{RoleArn: "some-arn"}:                      nil,
					{RoleArn: "some-arn", ExternalID: "thing"}: nil,
					{RoleArn: "some-arn2"}:                     nil,
					{RoleArn: "some-arn3"}:                     nil,
					{RoleArn: "some-arn4"}:                     nil,
				},
				clients: map[config.Role]map[string]*cachedClients{
					{RoleArn: "some-arn"}: {
						"ap-northeast-1": &cachedClients{onlyStatic: true},
						"eu-west-2":      &cachedClients{onlyStatic: true},
						"us-east-1":      &cachedClients{onlyStatic: true},
					},
					{RoleArn: "some-arn", ExternalID: "thing"}: {
						"us-east-1": &cachedClients{onlyStatic: true},
					},
					{RoleArn: "some-arn2"}: {
						"ap-northeast-1": &cachedClients{onlyStatic: true},
						"eu-west-2":      &cachedClients{onlyStatic: true},
						"us-east-1":      &cachedClients{onlyStatic: true},
					},
					{RoleArn: "some-arn3"}: {
						"eu-west-2": &cachedClients{onlyStatic: true},
						"us-east-1": &cachedClients{onlyStatic: true},
					},
					{RoleArn: "some-arn4"}: {
						"ap-northeast-1": &cachedClients{onlyStatic: true},
					},
				},
				logger: logging.NewNopLogger(),
			},
		},
	}

	for _, l := range tests {
		test := l
		t.Run(test.descrip, func(t *testing.T) {
			t.Parallel()
			cache := NewClientCache(test.config, test.fips, logging.NewNopLogger()).(*clientCache)
			t.Logf("the cache is: %v", cache)

			if test.cache.cleared != cache.cleared {
				t.Logf("`cleared` not equal got %v, expected %v", cache.cleared, test.cache.cleared)
				t.Fail()
			}

			if test.cache.refreshed != cache.refreshed {
				t.Logf("`refreshed` not equal got %v, expected %v", cache.refreshed, test.cache.refreshed)
				t.Fail()
			}

			if test.cache.fips != cache.fips {
				t.Logf("`fips` not equal got %v, expected %v", cache.fips, test.cache.fips)
				t.Fail()
			}

			// Strict equality requires each set containing each other
			cmpCache(t, test.cache, cache)
			cmpCache(t, cache, test.cache)
		})
	}
}

func TestClear(t *testing.T) {
	region := "us-east-1"
	role := config.Role{}

	tests := []struct {
		description string
		cache       *clientCache
	}{
		{
			"a new clear clears all clients",
			&clientCache{
				session: mock.Session,
				cleared: false,
				mu:      sync.Mutex{},
				stscache: map[config.Role]stsiface.STSAPI{
					{}: nil,
				},
				clients: map[config.Role]map[string]*cachedClients{
					{}: {
						"us-east-1": &cachedClients{
							cloudwatch: createCloudWatchClient(logging.NewNopLogger(), mock.Session, &region, role, false),
							tagging:    createTaggingClient(logging.NewNopLogger(), mock.Session, &region, role, false),
							account:    createAccountClient(logging.NewNopLogger(), nil),
							onlyStatic: true,
						},
					},
				},
				logger: logging.NewNopLogger(),
			},
		},
		{
			"A second call to clear does nothing",
			&clientCache{
				cleared: true,
				mu:      sync.Mutex{},
				session: mock.Session,
				stscache: map[config.Role]stsiface.STSAPI{
					{}: nil,
				},
				clients: map[config.Role]map[string]*cachedClients{
					{}: {
						"us-east-1": &cachedClients{
							cloudwatch: nil,
							tagging:    nil,
							account:    nil,
						},
					},
				},
				logger: logging.NewNopLogger(),
			},
		},
	}

	for _, l := range tests {
		test := l
		t.Run(test.description, func(t *testing.T) {
			test.cache.Clear()
			if !test.cache.cleared {
				t.Log("Cache cleared flag not set")
				t.Fail()
			}
			if test.cache.refreshed {
				t.Log("Cache cleared flag set")
				t.Fail()
			}

			for role, client := range test.cache.stscache {
				if client != nil {
					t.Logf("STS `client` %v not cleared", role)
					t.Fail()
				}
			}

			for role, regionMap := range test.cache.clients {
				for region, client := range regionMap {
					if client.cloudwatch != nil {
						t.Logf("`cloudwatch client` %v in region %v is not nil", role, region)
						t.Fail()
					}
					if client.tagging != nil {
						t.Logf("`tagging client` %v in region %v is not nil", role, region)
						t.Fail()
					}
					if client.account != nil {
						t.Logf("`asg client` %v in region %v is not nil", role, region)
						t.Fail()
					}
				}
			}
		})
	}
}

func TestRefresh(t *testing.T) {
	region := "us-east-1"
	role := config.Role{}

	tests := []struct {
		descrip    string
		cache      *clientCache
		cloudwatch bool
	}{
		{
			"a new refresh creates clients",
			&clientCache{
				session:   mock.Session,
				refreshed: false,
				mu:        sync.Mutex{},
				stscache: map[config.Role]stsiface.STSAPI{
					{}: nil,
				},
				clients: map[config.Role]map[string]*cachedClients{
					{}: {
						"us-east-1": &cachedClients{
							cloudwatch: nil,
							tagging:    nil,
							account:    nil,
						},
					},
				},
				logger: logging.NewNopLogger(),
			},
			false,
		},
		{
			"a new refresh with static only creates only cloudwatch",
			&clientCache{
				session:   mock.Session,
				refreshed: false,
				mu:        sync.Mutex{},
				stscache: map[config.Role]stsiface.STSAPI{
					{}: nil,
				},
				clients: map[config.Role]map[string]*cachedClients{
					{}: {
						"us-east-1": &cachedClients{
							cloudwatch: nil,
							tagging:    nil,
							account:    nil,
							onlyStatic: true,
						},
					},
				},
				logger: logging.NewNopLogger(),
			},
			true,
		},
		{
			"A second call to refreshed does nothing",
			&clientCache{
				refreshed: true,
				mu:        sync.Mutex{},
				session:   mock.Session,
				stscache: map[config.Role]stsiface.STSAPI{
					{}: createStsSession(mock.Session, role, "", false, false),
				},
				clients: map[config.Role]map[string]*cachedClients{
					{}: {
						"us-east-1": &cachedClients{
							cloudwatch: createCloudWatchClient(logging.NewNopLogger(), mock.Session, &region, role, false),
							tagging:    createTaggingClient(logging.NewNopLogger(), mock.Session, &region, role, false),
							account:    createAccountClient(logging.NewNopLogger(), createStsSession(mock.Session, role, "", false, false)),
						},
					},
				},
				logger: logging.NewNopLogger(),
			},
			false,
		},
	}

	for _, l := range tests {
		test := l
		t.Run(test.descrip, func(t *testing.T) {
			t.Parallel()
			test.cache.Refresh()

			if !test.cache.refreshed {
				t.Log("Cache refreshed flag not set")
				t.Fail()
			}

			if test.cache.cleared {
				t.Log("Cache cleared flag set")
				t.Fail()
			}

			for role, client := range test.cache.stscache {
				if client == nil {
					t.Logf("STS `client` %v not refreshed", role)
					t.Fail()
				}
			}

			for role, regionMap := range test.cache.clients {
				for region, client := range regionMap {
					if client.cloudwatch == nil {
						t.Logf("`cloudwatch client` %v in region %v still nil", role, region)
						t.Fail()
					}
					if test.cloudwatch {
						continue
					}
					if client.tagging == nil {
						t.Logf("`tagging client` %v in region %v still nil", role, region)
						t.Fail()
					}
					if client.account == nil {
						t.Logf("`asg client` %v in region %v still nil", role, region)
						t.Fail()
					}
				}
			}
		})
	}
}

func TestClientCacheGetCloudwatchClient(t *testing.T) {
	testGetAWSClient(
		t, "Cloudwatch",
		func(t *testing.T, cache *clientCache, region string, role config.Role) {
			iface := cache.GetCloudwatchClient(region, role, 1)
			if iface == nil {
				t.Fail()
				return
			}
		})
}

func TestClientCacheGetTagging(t *testing.T) {
	testGetAWSClient(
		t, "Tagging",
		func(t *testing.T, cache *clientCache, region string, role config.Role) {
			iface := cache.GetTaggingClient(region, role, 1)
			if iface == nil {
				t.Fail()
				return
			}
		})
}

func TestClientCacheGetAccount(t *testing.T) {
	testGetAWSClient(
		t, "Account",
		func(t *testing.T, cache *clientCache, region string, role config.Role) {
			iface := cache.GetAccountClient(region, role)
			if iface == nil {
				t.Fail()
				return
			}
		})
}

func testGetAWSClient(
	t *testing.T,
	name string,
	testClientGet func(*testing.T, *clientCache, string, config.Role),
) {
	region := "us-east-1"
	role := config.Role{}
	tests := []struct {
		descrip     string
		cache       *clientCache
		parallelRun bool
	}{
		{
			"locks during unrefreshed parallel call",
			&clientCache{
				refreshed: false,
				mu:        sync.Mutex{},
				session:   mock.Session,
				stscache: map[config.Role]stsiface.STSAPI{
					{}: nil,
				},
				clients: map[config.Role]map[string]*cachedClients{
					{}: {
						"us-east-1": &cachedClients{
							cloudwatch: createCloudWatchClient(logging.NewNopLogger(), mock.Session, &region, role, false),
							tagging:    createTaggingClient(logging.NewNopLogger(), mock.Session, &region, role, false),
							account:    createAccountClient(logging.NewNopLogger(), createStsSession(mock.Session, role, "", false, false)),
						},
					},
				},
				logger: logging.NewNopLogger(),
			},
			true,
		},
		{
			"returns clients if available",
			&clientCache{
				refreshed: true,
				session:   mock.Session,
				mu:        sync.Mutex{},
				stscache: map[config.Role]stsiface.STSAPI{
					{}: nil,
				},
				clients: map[config.Role]map[string]*cachedClients{
					{}: {
						"us-east-1": &cachedClients{
							cloudwatch: createCloudWatchClient(logging.NewNopLogger(), mock.Session, &region, role, false),
							tagging:    createTaggingClient(logging.NewNopLogger(), mock.Session, &region, role, false),
							account:    createAccountClient(logging.NewNopLogger(), createStsSession(mock.Session, role, "", false, false)),
						},
					},
				},
				logger: logging.NewNopLogger(),
			},
			false,
		},
		{
			"creates a new clients if not available",
			&clientCache{
				refreshed: true,
				session:   mock.Session,
				mu:        sync.Mutex{},
				stscache: map[config.Role]stsiface.STSAPI{
					{}: nil,
				},
				clients: map[config.Role]map[string]*cachedClients{
					{}: {
						"us-east-1": &cachedClients{},
					},
				},
				logger: logging.NewNopLogger(),
			},
			false,
		},
	}

	for _, l := range tests {
		test := l
		t.Run(name+" "+test.descrip, func(t *testing.T) {
			t.Parallel()
			if test.parallelRun {
				go testClientGet(t, test.cache, region, role)
			}
			testClientGet(t, test.cache, region, role)

			if test.cache.clients[role][region] == nil {
				t.Log("cache is nil when it should be populated")
				t.Fail()
			}
		})
	}
}

func TestSetExternalID(t *testing.T) {
	tests := []struct {
		descrip string
		ID      string
		isSet   bool
	}{
		{
			"sets the external ID if not empty",
			"should-be-set",
			true,
		},
		{
			"external ID not set if empty",
			"",
			false,
		},
	}

	for _, l := range tests {
		test := l
		t.Run(test.descrip, func(t *testing.T) {
			f := setExternalID(test.ID)
			p := &stscreds.AssumeRoleProvider{}
			f(p)
			if test.isSet {
				if *p.ExternalID != test.ID {
					t.Fail()
				}
			}
		})
	}
}

func TestSetSTSCreds(t *testing.T) {
	tests := []struct {
		descrip        string
		role           config.Role
		credentialsNil bool
		externalID     string
	}{
		{
			"sets the sts creds if the role arn is set",
			config.Role{
				RoleArn: "this:arn",
			},
			false,
			"",
		},
		{
			"does not set the creds if role arn is not set",
			config.Role{},
			true,
			"",
		},
		{
			"does not set the creds if role arn is not set & external id is set",
			config.Role{
				ExternalID: "thing",
			},
			true,
			"",
		},
	}

	for _, l := range tests {
		test := l
		t.Run(test.descrip, func(t *testing.T) {
			t.Parallel()
			conf := setSTSCreds(mock.Session, &aws.Config{}, test.role)
			if test.credentialsNil {
				if conf.Credentials != nil {
					t.Fail()
				}
			} else {
				if conf.Credentials == nil {
					t.Fail()
				}
			}
		})
	}
}

func TestCreateAWSSession(t *testing.T) {
	tests := []struct {
		descrip string
	}{
		{
			"creates an aws clients",
		},
	}

	for _, l := range tests {
		test := l
		t.Run(test.descrip, func(t *testing.T) {
			s := createAWSSession(endpoints.DefaultResolver().EndpointFor, false)
			if s == nil {
				t.Fail()
			}
		})
	}
}

func TestCreateStsSession(t *testing.T) {
	tests := []struct {
		descrip   string
		role      config.Role
		stsRegion string
	}{
		{
			"creates an sts clients with an empty role",
			config.Role{},
			"",
		},
		{
			"creates an sts clients with region",
			config.Role{},
			"eu-west-1",
		},
		{
			"creates an sts clients with an empty external id",
			config.Role{
				RoleArn: "some:arn",
			},
			"",
		},
		{
			"creates an sts clients with an empty role arn",
			config.Role{
				ExternalID: "some-id",
			},
			"",
		},
		{
			"creates an sts clients with an sts full role",
			config.Role{
				RoleArn:    "some:arn",
				ExternalID: "some-id",
			},
			"",
		},
	}

	for _, l := range tests {
		test := l
		t.Run(test.descrip, func(t *testing.T) {
			t.Parallel()
			// just exercise the code path
			iface := createStsSession(mock.Session, test.role, test.stsRegion, false, false)
			if iface == nil {
				t.Fail()
			}
		})
	}
}

func TestCreateCloudwatchSession(t *testing.T) {
	testAWSClient(
		t,
		"Cloudwatch",
		func(t *testing.T, s *session.Session, region *string, role config.Role, fips bool) {
			iface := createCloudwatchSession(s, region, role, fips, false)
			if iface == nil {
				t.Fail()
			}
		})
}

func TestCreateTagSession(t *testing.T) {
	testAWSClient(
		t,
		"Tag",
		func(t *testing.T, s *session.Session, region *string, role config.Role, fips bool) {
			iface := createTagSession(s, region, role, fips)
			if iface == nil {
				t.Fail()
			}
		})
}

func TestCreateASGSession(t *testing.T) {
	testAWSClient(
		t,
		"ASG",
		func(t *testing.T, s *session.Session, region *string, role config.Role, fips bool) {
			iface := createASGSession(s, region, role, fips)
			if iface == nil {
				t.Fail()
			}
		})
}

func TestCreateEC2Session(t *testing.T) {
	testAWSClient(
		t,
		"EC2",
		func(t *testing.T, s *session.Session, region *string, role config.Role, fips bool) {
			iface := createEC2Session(s, region, role, fips, false)
			if iface == nil {
				t.Fail()
			}
		})
}

func TestCreatePrometheusSession(t *testing.T) {
	testAWSClient(
		t,
		"Prometheus",
		func(t *testing.T, s *session.Session, region *string, role config.Role, fips bool) {
			iface := createPrometheusSession(s, region, role, fips, false)
			if iface == nil {
				t.Fail()
			}
		})
}

func TestCreateDMSSession(t *testing.T) {
	testAWSClient(
		t,
		"DMS",
		func(t *testing.T, s *session.Session, region *string, role config.Role, fips bool) {
			iface := createDMSSession(s, region, role, fips, false)
			if iface == nil {
				t.Fail()
			}
		})
}

func TestCreateAPIGatewaySession(t *testing.T) {
	testAWSClient(
		t,
		"APIGateway",
		func(t *testing.T, s *session.Session, region *string, role config.Role, fips bool) {
			iface := createAPIGatewaySession(s, region, role, fips, false)
			if iface == nil {
				t.Fail()
			}
		})
}

func TestCreateStorageGatewaySession(t *testing.T) {
	testAWSClient(
		t,
		"StorageGateway",
		func(t *testing.T, s *session.Session, region *string, role config.Role, fips bool) {
			iface := createStorageGatewaySession(s, region, role, fips, false)
			if iface == nil {
				t.Fail()
			}
		})
}

func testAWSClient(
	t *testing.T,
	name string,
	testClientCreation func(*testing.T, *session.Session, *string, config.Role, bool),
) {
	tests := []struct {
		descrip string
		region  string
		role    config.Role
		fips    bool
	}{
		{
			fmt.Sprintf("%s client without role and fips is created", name),
			"us-east-1",
			config.Role{},
			false,
		},
		{
			fmt.Sprintf("%s client without role and with fips is created", name),
			"us-east-1",
			config.Role{},
			true,
		},
		{
			fmt.Sprintf("%s client with roleARN and without external id is created", name),
			"us-east-1",
			config.Role{
				RoleArn: "some:arn",
			},
			false,
		},
		{
			fmt.Sprintf("%s client with roleARN and with external id is created", name),
			"us-east-1",
			config.Role{
				RoleArn:    "some:arn",
				ExternalID: "some-id",
			},
			false,
		},
		{
			fmt.Sprintf("%s client without roleARN and with external id is created", name),
			"us-east-1",
			config.Role{
				ExternalID: "some-id",
			},
			false,
		},
	}

	for _, l := range tests {
		test := l
		t.Run(test.descrip, func(t *testing.T) {
			t.Parallel()
			testClientCreation(t, mock.Session, &test.region, test.role, test.fips)
		})
	}
}
