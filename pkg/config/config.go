package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"gopkg.in/yaml.v2"

	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/logging"
	"github.com/nerdswords/yet-another-cloudwatch-exporter/pkg/model"
)

type ScrapeConf struct {
	APIVersion      string             `yaml:"apiVersion"`
	StsRegion       string             `yaml:"sts-region"`
	Discovery       Discovery          `yaml:"discovery"`
	Static          []*Static          `yaml:"static"`
	CustomNamespace []*CustomNamespace `yaml:"customNamespace"`
}

type Discovery struct {
	ExportedTagsOnMetrics ExportedTagsOnMetrics `yaml:"exportedTagsOnMetrics"`
	Jobs                  []*Job                `yaml:"jobs"`
}

type ExportedTagsOnMetrics map[string][]string

type Tag struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

type JobLevelMetricFields struct {
	Statistics             []string `yaml:"statistics"`
	Period                 int64    `yaml:"period"`
	Length                 int64    `yaml:"length"`
	Delay                  int64    `yaml:"delay"`
	NilToZero              *bool    `yaml:"nilToZero"`
	AddCloudwatchTimestamp *bool    `yaml:"addCloudwatchTimestamp"`
}

type Job struct {
	Regions                     []string  `yaml:"regions"`
	Type                        string    `yaml:"type"`
	Roles                       []Role    `yaml:"roles"`
	SearchTags                  []Tag     `yaml:"searchTags"`
	CustomTags                  []Tag     `yaml:"customTags"`
	DimensionNameRequirements   []string  `yaml:"dimensionNameRequirements"`
	Metrics                     []*Metric `yaml:"metrics"`
	RoundingPeriod              *int64    `yaml:"roundingPeriod"`
	RecentlyActiveOnly          bool      `yaml:"recentlyActiveOnly"`
	IncludeContextOnInfoMetrics bool      `yaml:"includeContextOnInfoMetrics"`
	JobLevelMetricFields        `yaml:",inline"`
}

type Static struct {
	Name       string      `yaml:"name"`
	Regions    []string    `yaml:"regions"`
	Roles      []Role      `yaml:"roles"`
	Namespace  string      `yaml:"namespace"`
	CustomTags []Tag       `yaml:"customTags"`
	Dimensions []Dimension `yaml:"dimensions"`
	Metrics    []*Metric   `yaml:"metrics"`
}

type CustomNamespace struct {
	Regions                   []string  `yaml:"regions"`
	Name                      string    `yaml:"name"`
	Namespace                 string    `yaml:"namespace"`
	RecentlyActiveOnly        bool      `yaml:"recentlyActiveOnly"`
	Roles                     []Role    `yaml:"roles"`
	Metrics                   []*Metric `yaml:"metrics"`
	CustomTags                []Tag     `yaml:"customTags"`
	DimensionNameRequirements []string  `yaml:"dimensionNameRequirements"`
	RoundingPeriod            *int64    `yaml:"roundingPeriod"`
	JobLevelMetricFields      `yaml:",inline"`
}

type Metric struct {
	Name                   string   `yaml:"name"`
	Statistics             []string `yaml:"statistics"`
	Period                 int64    `yaml:"period"`
	Length                 int64    `yaml:"length"`
	Delay                  int64    `yaml:"delay"`
	NilToZero              *bool    `yaml:"nilToZero"`
	AddCloudwatchTimestamp *bool    `yaml:"addCloudwatchTimestamp"`
}

type Dimension struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type Role struct {
	RoleArn    string `yaml:"roleArn"`
	ExternalID string `yaml:"externalId"`
}

func (r *Role) ValidateRole(roleIdx int, parent string) error {
	if r.RoleArn == "" && r.ExternalID != "" {
		return fmt.Errorf("Role [%d] in %v: RoleArn should not be empty", roleIdx, parent)
	}

	return nil
}

func (c *ScrapeConf) Load(file string, logger logging.Logger) (model.JobsConfig, error) {
	yamlFile, err := os.ReadFile(file)
	if err != nil {
		return model.JobsConfig{}, err
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		return model.JobsConfig{}, err
	}

	logConfigErrors(yamlFile, logger)

	for _, job := range c.Discovery.Jobs {
		if len(job.Roles) == 0 {
			job.Roles = []Role{{}} // use current IAM role
		}
	}

	for _, job := range c.CustomNamespace {
		if len(job.Roles) == 0 {
			job.Roles = []Role{{}} // use current IAM role
		}
	}

	for _, job := range c.Static {
		if len(job.Roles) == 0 {
			job.Roles = []Role{{}} // use current IAM role
		}
	}

	return c.Validate()
}

func (c *ScrapeConf) Validate() (model.JobsConfig, error) {
	if c.Discovery.Jobs == nil && c.Static == nil && c.CustomNamespace == nil {
		return model.JobsConfig{}, fmt.Errorf("At least 1 Discovery job, 1 Static or one CustomNamespace must be defined")
	}

	if c.Discovery.Jobs != nil {
		for idx, job := range c.Discovery.Jobs {
			err := job.validateDiscoveryJob(idx)
			if err != nil {
				return model.JobsConfig{}, err
			}
		}
	}

	if c.CustomNamespace != nil {
		for idx, job := range c.CustomNamespace {
			err := job.validateCustomNamespaceJob(idx)
			if err != nil {
				return model.JobsConfig{}, err
			}
		}
	}

	if c.Static != nil {
		for idx, job := range c.Static {
			err := job.validateStaticJob(idx)
			if err != nil {
				return model.JobsConfig{}, err
			}
		}
	}
	if c.APIVersion != "" && c.APIVersion != "v1alpha1" {
		return model.JobsConfig{}, fmt.Errorf("unknown apiVersion value '%s'", c.APIVersion)
	}

	return c.toModelConfig(), nil
}

func (j *Job) validateDiscoveryJob(jobIdx int) error {
	if j.Type != "" {
		if SupportedServices.GetService(j.Type) == nil {
			return fmt.Errorf("Discovery job [%d]: Service is not in known list!: %s", jobIdx, j.Type)
		}
	} else {
		return fmt.Errorf("Discovery job [%d]: Type should not be empty", jobIdx)
	}
	parent := fmt.Sprintf("Discovery job [%s/%d]", j.Type, jobIdx)
	if len(j.Roles) > 0 {
		for roleIdx, role := range j.Roles {
			if err := role.ValidateRole(roleIdx, parent); err != nil {
				return err
			}
		}
	} else {
		return fmt.Errorf("no IAM roles configured. If the current IAM role is desired, an empty Role should be configured")
	}
	if len(j.Regions) == 0 {
		return fmt.Errorf("Discovery job [%s/%d]: Regions should not be empty", j.Type, jobIdx)
	}
	if len(j.Metrics) == 0 {
		return fmt.Errorf("Discovery job [%s/%d]: Metrics should not be empty", j.Type, jobIdx)
	}
	for metricIdx, metric := range j.Metrics {
		err := metric.validateMetric(metricIdx, parent, &j.JobLevelMetricFields)
		if err != nil {
			return err
		}
	}

	return nil
}

func (j *CustomNamespace) validateCustomNamespaceJob(jobIdx int) error {
	if j.Name == "" {
		return fmt.Errorf("CustomNamespace job [%v]: Name should not be empty", jobIdx)
	}
	if j.Namespace == "" {
		return fmt.Errorf("CustomNamespace job [%v]: Namespace should not be empty", jobIdx)
	}
	parent := fmt.Sprintf("CustomNamespace job [%s/%d]", j.Namespace, jobIdx)
	if len(j.Roles) > 0 {
		for roleIdx, role := range j.Roles {
			if err := role.ValidateRole(roleIdx, parent); err != nil {
				return err
			}
		}
	} else {
		return fmt.Errorf("no IAM roles configured. If the current IAM role is desired, an empty Role should be configured")
	}
	if j.Regions == nil || len(j.Regions) == 0 {
		return fmt.Errorf("CustomNamespace job [%s/%d]: Regions should not be empty", j.Name, jobIdx)
	}
	if len(j.Metrics) == 0 {
		return fmt.Errorf("CustomNamespace job [%s/%d]: Metrics should not be empty", j.Name, jobIdx)
	}
	for metricIdx, metric := range j.Metrics {
		err := metric.validateMetric(metricIdx, parent, &j.JobLevelMetricFields)
		if err != nil {
			return err
		}
	}

	return nil
}

func (j *Static) validateStaticJob(jobIdx int) error {
	if j.Name == "" {
		return fmt.Errorf("Static job [%v]: Name should not be empty", jobIdx)
	}
	if j.Namespace == "" {
		return fmt.Errorf("Static job [%s/%d]: Namespace should not be empty", j.Name, jobIdx)
	}
	parent := fmt.Sprintf("Static job [%s/%d]", j.Name, jobIdx)
	if len(j.Roles) > 0 {
		for roleIdx, role := range j.Roles {
			if err := role.ValidateRole(roleIdx, parent); err != nil {
				return err
			}
		}
	} else {
		return fmt.Errorf("no IAM roles configured. If the current IAM role is desired, an empty Role should be configured")
	}
	if len(j.Regions) == 0 {
		return fmt.Errorf("Static job [%s/%d]: Regions should not be empty", j.Name, jobIdx)
	}
	for metricIdx, metric := range j.Metrics {
		err := metric.validateMetric(metricIdx, parent, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Metric) validateMetric(metricIdx int, parent string, discovery *JobLevelMetricFields) error {
	if m.Name == "" {
		return fmt.Errorf("Metric [%s/%d] in %v: Name should not be empty", m.Name, metricIdx, parent)
	}

	mStatistics := m.Statistics
	if len(mStatistics) == 0 && discovery != nil {
		if len(discovery.Statistics) > 0 {
			mStatistics = discovery.Statistics
		} else {
			return fmt.Errorf("Metric [%s/%d] in %v: Statistics should not be empty", m.Name, metricIdx, parent)
		}
	}

	mPeriod := m.Period
	if mPeriod == 0 {
		if discovery != nil && discovery.Period != 0 {
			mPeriod = discovery.Period
		} else {
			mPeriod = model.DefaultPeriodSeconds
		}
	}
	if mPeriod < 1 {
		return fmt.Errorf("Metric [%s/%d] in %v: Period value should be a positive integer", m.Name, metricIdx, parent)
	}
	mLength := m.Length
	if mLength == 0 {
		if discovery != nil && discovery.Length != 0 {
			mLength = discovery.Length
		} else {
			mLength = model.DefaultLengthSeconds
		}
	}

	mDelay := m.Delay
	if mDelay == 0 {
		if discovery != nil && discovery.Delay != 0 {
			mDelay = discovery.Delay
		} else {
			mDelay = model.DefaultDelaySeconds
		}
	}

	mNilToZero := m.NilToZero
	if mNilToZero == nil {
		if discovery != nil && discovery.NilToZero != nil {
			mNilToZero = discovery.NilToZero
		} else {
			mNilToZero = aws.Bool(false)
		}
	}

	mAddCloudwatchTimestamp := m.AddCloudwatchTimestamp
	if mAddCloudwatchTimestamp == nil {
		if discovery != nil && discovery.AddCloudwatchTimestamp != nil {
			mAddCloudwatchTimestamp = discovery.AddCloudwatchTimestamp
		} else {
			mAddCloudwatchTimestamp = aws.Bool(false)
		}
	}

	if mLength < mPeriod {
		return fmt.Errorf(
			"Metric [%s/%d] in %v: length(%d) is smaller than period(%d). This can cause that the data requested is not ready and generate data gaps",
			m.Name, metricIdx, parent, mLength, mPeriod,
		)
	}
	m.Length = mLength
	m.Period = mPeriod
	m.Delay = mDelay
	m.NilToZero = mNilToZero
	m.AddCloudwatchTimestamp = mAddCloudwatchTimestamp
	m.Statistics = mStatistics

	return nil
}

func (c *ScrapeConf) toModelConfig() model.JobsConfig {
	jobsCfg := model.JobsConfig{}
	jobsCfg.StsRegion = c.StsRegion

	for _, discoveryJob := range c.Discovery.Jobs {
		job := model.DiscoveryJob{}
		job.Regions = discoveryJob.Regions
		job.Type = discoveryJob.Type
		job.DimensionNameRequirements = discoveryJob.DimensionNameRequirements
		job.RoundingPeriod = discoveryJob.RoundingPeriod
		job.RecentlyActiveOnly = discoveryJob.RecentlyActiveOnly
		job.Statistics = discoveryJob.Statistics
		job.Period = discoveryJob.Period
		job.Length = discoveryJob.Length
		job.Delay = discoveryJob.Delay
		job.NilToZero = discoveryJob.NilToZero
		job.AddCloudwatchTimestamp = discoveryJob.AddCloudwatchTimestamp
		job.Roles = toModelRoles(discoveryJob.Roles)
		job.SearchTags = toModelTags(discoveryJob.SearchTags)
		job.CustomTags = toModelTags(discoveryJob.CustomTags)
		job.Metrics = toModelMetricConfig(discoveryJob.Metrics)
		job.IncludeContextOnInfoMetrics = discoveryJob.IncludeContextOnInfoMetrics

		job.ExportedTagsOnMetrics = []string{}
		if len(c.Discovery.ExportedTagsOnMetrics) > 0 {
			svc := SupportedServices.GetService(job.Type)
			if exportedTags, ok := c.Discovery.ExportedTagsOnMetrics[svc.Namespace]; ok {
				job.ExportedTagsOnMetrics = exportedTags
			} else if exportedTags, ok := c.Discovery.ExportedTagsOnMetrics[svc.Alias]; ok {
				job.ExportedTagsOnMetrics = exportedTags
			}
		}

		jobsCfg.DiscoveryJobs = append(jobsCfg.DiscoveryJobs, job)
	}

	for _, staticJob := range c.Static {
		job := model.StaticJob{}
		job.Name = staticJob.Name
		job.Namespace = staticJob.Namespace
		job.Regions = staticJob.Regions
		job.Roles = toModelRoles(staticJob.Roles)
		job.CustomTags = toModelTags(staticJob.CustomTags)
		job.Dimensions = toModelDimensions(staticJob.Dimensions)
		job.Metrics = toModelMetricConfig(staticJob.Metrics)
		jobsCfg.StaticJobs = append(jobsCfg.StaticJobs, job)
	}

	for _, customNamespaceJob := range c.CustomNamespace {
		job := model.CustomNamespaceJob{}
		job.Regions = customNamespaceJob.Regions
		job.Name = customNamespaceJob.Name
		job.Namespace = customNamespaceJob.Namespace
		job.DimensionNameRequirements = customNamespaceJob.DimensionNameRequirements
		job.RoundingPeriod = customNamespaceJob.RoundingPeriod
		job.RecentlyActiveOnly = customNamespaceJob.RecentlyActiveOnly
		job.Statistics = customNamespaceJob.Statistics
		job.Period = customNamespaceJob.Period
		job.Length = customNamespaceJob.Length
		job.Delay = customNamespaceJob.Delay
		job.NilToZero = customNamespaceJob.NilToZero
		job.AddCloudwatchTimestamp = customNamespaceJob.AddCloudwatchTimestamp
		job.Roles = toModelRoles(customNamespaceJob.Roles)
		job.CustomTags = toModelTags(customNamespaceJob.CustomTags)
		job.Metrics = toModelMetricConfig(customNamespaceJob.Metrics)
		jobsCfg.CustomNamespaceJobs = append(jobsCfg.CustomNamespaceJobs, job)
	}

	return jobsCfg
}

func toModelTags(tags []Tag) []model.Tag {
	ret := make([]model.Tag, 0, len(tags))
	for _, t := range tags {
		ret = append(ret, model.Tag{
			Key:   t.Key,
			Value: t.Value,
		})
	}
	return ret
}

func toModelRoles(roles []Role) []model.Role {
	ret := make([]model.Role, 0, len(roles))
	for _, r := range roles {
		ret = append(ret, model.Role{
			RoleArn:    r.RoleArn,
			ExternalID: r.ExternalID,
		})
	}
	return ret
}

func toModelDimensions(dimensions []Dimension) []model.Dimension {
	ret := make([]model.Dimension, 0, len(dimensions))
	for _, d := range dimensions {
		ret = append(ret, model.Dimension{
			Name:  d.Name,
			Value: d.Value,
		})
	}
	return ret
}

func toModelMetricConfig(metrics []*Metric) []*model.MetricConfig {
	ret := make([]*model.MetricConfig, 0, len(metrics))
	for _, m := range metrics {
		ret = append(ret, &model.MetricConfig{
			Name:                   m.Name,
			Statistics:             m.Statistics,
			Period:                 m.Period,
			Length:                 m.Length,
			Delay:                  m.Delay,
			NilToZero:              m.NilToZero,
			AddCloudwatchTimestamp: m.AddCloudwatchTimestamp,
		})
	}
	return ret
}

// logConfigErrors logs as warning any config unmarshalling error.
func logConfigErrors(cfg []byte, logger logging.Logger) {
	var sc ScrapeConf
	var errMsgs []string
	if err := yaml.UnmarshalStrict(cfg, &sc); err != nil {
		terr := &yaml.TypeError{}
		if errors.As(err, &terr) {
			errMsgs = append(errMsgs, terr.Errors...)
		} else {
			errMsgs = append(errMsgs, err.Error())
		}
	}

	if sc.APIVersion == "" {
		errMsgs = append(errMsgs, "missing apiVersion")
	}

	if len(errMsgs) > 0 {
		for _, msg := range errMsgs {
			logger.Warn("config file syntax error", "err", msg)
		}
		logger.Warn(`Config file error(s) detected: Yace might not work as expected. Future versions of Yace might fail to run with an invalid config file.`)
	}
}
