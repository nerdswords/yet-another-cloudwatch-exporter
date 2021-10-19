package exporter

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/client"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/apigateway/apigatewayiface"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	r "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	log "github.com/sirupsen/logrus"
)

// SessionCache is an interface to a cache of sessions and clients for all the
// roles specified by the exporter. For jobs with many duplicate roles, this provides
// relief to the AWS API and prevents timeouts by excessive credential requesting.
type SessionCache interface {
	GetSTS(Role) stsiface.STSAPI
	GetCloudwatch(*string, Role) cloudwatchiface.CloudWatchAPI
	GetTagging(*string, Role) resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI
	GetASG(*string, Role) autoscalingiface.AutoScalingAPI
	GetEC2(*string, Role) ec2iface.EC2API
	GetAPIGateway(*string, Role) apigatewayiface.APIGatewayAPI
	Refresh()
	Clear()
}

type sessionCache struct {
	session   *session.Session
	stscache  map[Role]stsiface.STSAPI
	clients   map[Role]map[string]*clientCache
	cleared   bool
	refreshed bool
	mu        sync.Mutex
	fips      bool
}

type clientCache struct {
	// if we know that this job is only used for static
	// then we don't have to construct as many cached connections
	// later on
	onlyStatic bool
	cloudwatch cloudwatchiface.CloudWatchAPI
	tagging    resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI
	asg        autoscalingiface.AutoScalingAPI
	ec2        ec2iface.EC2API
	apiGateway apigatewayiface.APIGatewayAPI
}

// NewSessionCache creates a new session cache to use when fetching data from
// AWS.
func NewSessionCache(config ScrapeConf, fips bool) SessionCache {
	stscache := map[Role]stsiface.STSAPI{}
	roleCache := map[Role]map[string]*clientCache{}

	for _, discoveryJob := range config.Discovery.Jobs {
		for _, role := range discoveryJob.Roles {
			if _, ok := stscache[role]; !ok {
				stscache[role] = nil
			}
			if _, ok := roleCache[role]; !ok {
				roleCache[role] = map[string]*clientCache{}
			}
			for _, region := range discoveryJob.Regions {
				roleCache[role][region] = &clientCache{}
			}
		}
	}

	for _, staticJob := range config.Static {
		for _, role := range staticJob.Roles {
			if _, ok := stscache[role]; !ok {
				stscache[role] = nil
			}

			if _, ok := roleCache[role]; !ok {
				roleCache[role] = map[string]*clientCache{}
			}

			for _, region := range staticJob.Regions {
				// Only write a new region in if the region does not exist
				if _, ok := roleCache[role][region]; !ok {
					roleCache[role][region] = &clientCache{
						onlyStatic: true,
					}
				}
			}
		}
	}

	return &sessionCache{
		session:   nil,
		stscache:  stscache,
		clients:   roleCache,
		fips:      fips,
		cleared:   false,
		refreshed: false,
	}
}

// Refresh and Clear help to avoid using lock primitives by asserting that
// there are no ongoing writes to the map.
func (s *sessionCache) Clear() {
	if s.cleared {
		return
	}

	for role := range s.stscache {
		s.stscache[role] = nil
	}

	for role, regions := range s.clients {
		for region := range regions {
			s.clients[role][region].cloudwatch = nil
			s.clients[role][region].tagging = nil
			s.clients[role][region].asg = nil
			s.clients[role][region].ec2 = nil
			s.clients[role][region].apiGateway = nil
		}
	}
	s.cleared = true
	s.refreshed = false
}

func (s *sessionCache) Refresh() {
	// TODO: make all the getter functions atomic pointer loads and sets
	if s.refreshed {
		return
	}

	// sessions really only need to be constructed once at runtime
	if s.session == nil {
		s.session = createAWSSession()
	}

	for role := range s.stscache {
		s.stscache[role] = createStsSession(s.session, role)
	}

	for role, regions := range s.clients {
		for region := range regions {
			// if the role is just used in static jobs, then we
			// can skip creating other sessions and potentially running
			// into permissions errors or taking up needless cycles
			s.clients[role][region].cloudwatch = createCloudwatchSession(s.session, &region, role, s.fips)
			if s.clients[role][region].onlyStatic {
				continue
			}

			s.clients[role][region].tagging = createTagSession(s.session, &region, role, s.fips)
			s.clients[role][region].asg = createASGSession(s.session, &region, role, s.fips)
			s.clients[role][region].ec2 = createEC2Session(s.session, &region, role, s.fips)
			s.clients[role][region].apiGateway = createAPIGatewaySession(s.session, &region, role, s.fips)
		}
	}

	s.cleared = false
	s.refreshed = true
}

func (s *sessionCache) GetSTS(role Role) stsiface.STSAPI {
	// if we have not refreshed then we need to lock in case we are accessing concurrently
	if !s.refreshed {
		s.mu.Lock()
		defer s.mu.Unlock()
	}
	if sess, ok := s.stscache[role]; ok && sess != nil {
		return sess
	}
	s.stscache[role] = createStsSession(s.session, role)
	return s.stscache[role]
}

func (s *sessionCache) GetCloudwatch(region *string, role Role) cloudwatchiface.CloudWatchAPI {
	// if we have not refreshed then we need to lock in case we are accessing concurrently
	if !s.refreshed {
		s.mu.Lock()
		defer s.mu.Unlock()
	}
	if sess, ok := s.clients[role][*region]; ok && sess.cloudwatch != nil {
		return sess.cloudwatch
	}
	s.clients[role][*region].cloudwatch = createCloudwatchSession(s.session, region, role, s.fips)
	return s.clients[role][*region].cloudwatch
}

func (s *sessionCache) GetTagging(region *string, role Role) resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI {
	// if we have not refreshed then we need to lock in case we are accessing concurrently
	if !s.refreshed {
		s.mu.Lock()
		defer s.mu.Unlock()
	}
	if sess, ok := s.clients[role][*region]; ok && sess.tagging != nil {
		return sess.tagging
	}

	s.clients[role][*region].tagging = createTagSession(s.session, region, role, s.fips)
	return s.clients[role][*region].tagging
}

func (s *sessionCache) GetASG(region *string, role Role) autoscalingiface.AutoScalingAPI {
	// if we have not refreshed then we need to lock in case we are accessing concurrently
	if !s.refreshed {
		s.mu.Lock()
		defer s.mu.Unlock()
	}
	if sess, ok := s.clients[role][*region]; ok && sess.asg != nil {
		return sess.asg
	}

	s.clients[role][*region].asg = createASGSession(s.session, region, role, s.fips)
	return s.clients[role][*region].asg
}

func (s *sessionCache) GetEC2(region *string, role Role) ec2iface.EC2API {
	// if we have not refreshed then we need to lock in case we are accessing concurrently
	if !s.refreshed {
		s.mu.Lock()
		defer s.mu.Unlock()
	}
	if sess, ok := s.clients[role][*region]; ok && sess.ec2 != nil {
		return sess.ec2
	}

	s.clients[role][*region].ec2 = createEC2Session(s.session, region, role, s.fips)
	return s.clients[role][*region].ec2
}

func (s *sessionCache) GetAPIGateway(region *string, role Role) apigatewayiface.APIGatewayAPI {
	// if we have not refreshed then we need to lock in case we are accessing concurrently
	if !s.refreshed {
		s.mu.Lock()
		defer s.mu.Unlock()
	}
	if sess, ok := s.clients[role][*region]; ok && sess.apiGateway != nil {
		return sess.apiGateway
	}

	s.clients[role][*region].apiGateway = createAPIGatewaySession(s.session, region, role, s.fips)
	return s.clients[role][*region].apiGateway

}

func setExternalID(ID string) func(p *stscreds.AssumeRoleProvider) {
	return func(p *stscreds.AssumeRoleProvider) {
		if ID != "" {
			p.ExternalID = aws.String(ID)
		}
	}
}

func setSTSCreds(sess *session.Session, config *aws.Config, role Role) *aws.Config {
	if role.RoleArn != "" {
		config.Credentials = stscreds.NewCredentials(
			sess, role.RoleArn, setExternalID(role.ExternalID))
	}
	return config
}

func getAwsRetryer() aws.RequestRetryer {
	return client.DefaultRetryer{
		NumMaxRetries: 5,
		// MaxThrottleDelay and MinThrottleDelay used for throttle errors
		MaxThrottleDelay: 10*time.Second,
		MinThrottleDelay: 1*time.Second,
		// For other errors
		MaxRetryDelay: 3*time.Second,
		MinRetryDelay: 1*time.Second,
	}
}

func createAWSSession() *session.Session {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config: aws.Config{
			CredentialsChainVerboseErrors: aws.Bool(true),
		},
	}))
	return sess
}

func createStsSession(sess *session.Session, role Role) *sts.STS {
	maxStsRetries := 5
	config := &aws.Config{MaxRetries: &maxStsRetries}
	if log.IsLevelEnabled(log.DebugLevel) {
		config.LogLevel = aws.LogLevel(aws.LogDebugWithHTTPBody)
	}
	return sts.New(sess, setSTSCreds(sess, config, role))
}

func createCloudwatchSession(sess *session.Session, region *string, role Role, fips bool) *cloudwatch.CloudWatch {

	config := &aws.Config{Region: region, Retryer: getAwsRetryer()}

	if fips {
		// https://docs.aws.amazon.com/general/latest/gr/cw_region.html
		endpoint := fmt.Sprintf("https://monitoring-fips.%s.amazonaws.com", *region)
		config.Endpoint = aws.String(endpoint)
	}

	if log.IsLevelEnabled(log.DebugLevel) {
		config.LogLevel = aws.LogLevel(aws.LogDebugWithHTTPBody)
	}

	return cloudwatch.New(sess, setSTSCreds(sess, config, role))
}

func createTagSession(sess *session.Session, region *string, role Role, fips bool) *r.ResourceGroupsTaggingAPI {
	maxResourceGroupTaggingRetries := 5
	config := &aws.Config{
		Region:                        region,
		MaxRetries:                    &maxResourceGroupTaggingRetries,
		CredentialsChainVerboseErrors: aws.Bool(true),
	}

	// ToDo: Resource Groups Tagging API does not have FIPS compliant endpoints
	// if fips {
	// 	https://docs.aws.amazon.com/general/latest/gr/arg.html
	// 	endpoint := fmt.Sprintf("https://tagging-fips.%s.amazonaws.com", *region)
	// 	config.Endpoint = aws.String(endpoint)
	// }

	return r.New(sess, setSTSCreds(sess, config, role))
}

func createASGSession(sess *session.Session, region *string, role Role, fips bool) autoscalingiface.AutoScalingAPI {
	maxAutoScalingAPIRetries := 5
	config := &aws.Config{Region: region, MaxRetries: &maxAutoScalingAPIRetries}

	// ToDo: Autoscaling does not have a FIPS endpoint
	// if fips {
	//   https://docs.aws.amazon.com/general/latest/gr/autoscaling_region.html
	//   endpoint := fmt.Sprintf("https://autoscaling-plans-fips.%s.amazonaws.com", *region)
	//   config.Endpoint = aws.String(endpoint)
	// }

	return autoscaling.New(sess, setSTSCreds(sess, config, role))
}

func createEC2Session(sess *session.Session, region *string, role Role, fips bool) ec2iface.EC2API {
	maxEC2APIRetries := 10
	config := &aws.Config{Region: region, MaxRetries: &maxEC2APIRetries}
	if fips {
		// https://docs.aws.amazon.com/general/latest/gr/ec2-service.html
		endpoint := fmt.Sprintf("https://ec2-fips.%s.amazonaws.com", *region)
		config.Endpoint = aws.String(endpoint)
	}

	return ec2.New(sess, setSTSCreds(sess, config, role))
}

func createAPIGatewaySession(sess *session.Session, region *string, role Role, fips bool) apigatewayiface.APIGatewayAPI {
	maxAPIGatewayAPIRetries := 5
	config := &aws.Config{Region: region, MaxRetries: &maxAPIGatewayAPIRetries}
	if fips {
		// https://docs.aws.amazon.com/general/latest/gr/apigateway.html
		endpoint := fmt.Sprintf("https://apigateway-fips.%s.amazonaws.com", *region)
		config.Endpoint = aws.String(endpoint)
	}

	return apigateway.New(sess, setSTSCreds(sess, config, role))
}
