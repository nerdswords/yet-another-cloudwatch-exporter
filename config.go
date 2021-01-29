package main

import (
	"fmt"
	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type conf struct {
	Discovery discovery `yaml:"discovery"`
	Static    []*static `yaml:"static"`
}

type discovery struct {
	ExportedTagsOnMetrics exportedTagsOnMetrics `yaml:"exportedTagsOnMetrics"`
	Jobs                  []*job                `yaml:"jobs"`
}

type exportedTagsOnMetrics map[string][]string

type job struct {
	Regions                []string  `yaml:"regions"`
	Namespace              string    `yaml:"namespace"`
	RoleArns               []string  `yaml:"roleArns"`
	SearchTags             []tag     `yaml:"searchTags"`
	CustomTags             []tag     `yaml:"customTags"`
	Metrics                []*metric `yaml:"metrics"`
	Length                 int       `yaml:"length"`
	Delay                  int       `yaml:"delay"`
	Period                 int       `yaml:"period"`
	AddCloudwatchTimestamp *bool     `yaml:"addCloudwatchTimestamp"`
	NilToZero              *bool     `yaml:"nilToZero"`
}

type static struct {
	Name       string      `yaml:"name"`
	Regions    []string    `yaml:"regions"`
	RoleArns   []string    `yaml:"roleArns"`
	Namespace  string      `yaml:"namespace"`
	CustomTags []tag       `yaml:"customTags"`
	Dimensions []dimension `yaml:"dimensions"`
	Metrics    []*metric   `yaml:"metrics"`
}

type metric struct {
	Name                   string   `yaml:"name"`
	Statistics             []string `yaml:"statistics"`
	Period                 int      `yaml:"period"`
	Length                 int      `yaml:"length"`
	Delay                  int      `yaml:"delay"`
	NilToZero              *bool    `yaml:"nilToZero"`
	AddCloudwatchTimestamp *bool    `yaml:"addCloudwatchTimestamp"`
}

type dimension struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type tag struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

func (c *conf) load(file *string) error {
	yamlFile, err := ioutil.ReadFile(*file)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		return err
	}

	for n, job := range c.Discovery.Jobs {
		if len(job.RoleArns) == 0 {
			c.Discovery.Jobs[n].RoleArns = []string{""} // use current IAM role
		}
	}
	for n, job := range c.Static {
		if len(job.RoleArns) == 0 {
			c.Static[n].RoleArns = []string{""} // use current IAM role
		}
	}

	err = c.validate()
	if err != nil {
		return err
	}
	return nil
}

func (c *conf) validate() error {
	if c.Discovery.Jobs == nil && c.Static == nil {
		return fmt.Errorf("At least 1 Discovery job or 1 Static must be defined")
	}

	if c.Discovery.Jobs != nil {
		for idx, job := range c.Discovery.Jobs {
			err := job.validateDiscoveryJob(idx)
			if err != nil {
				return err
			}
		}
	}

	if c.Static != nil {
		for idx, job := range c.Static {
			err := job.validateStaticJob(idx)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (j *job) validateDiscoveryJob(jobIdx int) error {
	if j.Namespace != "" {
		if _, ok := supportedServices[j.Namespace]; !ok {
			return fmt.Errorf("Discovery job [%d]: Service is not in known list!: %s", jobIdx, j.Namespace)
		}
	} else {
		return fmt.Errorf("Discovery job [%d]: Namespace should not be empty", jobIdx)
	}
	if len(j.Regions) == 0 {
		return fmt.Errorf("Discovery job [%s/%d]: Regions should not be empty", j.Namespace, jobIdx)
	}
	if len(j.Metrics) == 0 {
		return fmt.Errorf("Discovery job [%s/%d]: Metrics should not be empty", j.Namespace, jobIdx)
	}
	for metricIdx, metric := range j.Metrics {
		parent := fmt.Sprintf("Discovery job [%s/%d]", j.Namespace, jobIdx)
		err := metric.validateMetric(metricIdx, parent, j)
		if err != nil {
			return err
		}
	}

	return nil
}

func (j *static) validateStaticJob(jobIdx int) error {
	if j.Name == "" {
		return fmt.Errorf("Static job [%v]: Name should not be empty", jobIdx)
	}
	if j.Namespace == "" {
		return fmt.Errorf("Static job [%s/%d]: Namespace should not be empty", j.Name, jobIdx)
	}
	if len(j.Regions) == 0 {
		return fmt.Errorf("Static job [%s/%d]: Regions should not be empty", j.Name, jobIdx)
	}
	for metricIdx, metric := range j.Metrics {
		err := metric.validateMetric(metricIdx, fmt.Sprintf("Static job [%s/%d]", j.Name, jobIdx), nil)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *metric) validateMetric(metricIdx int, parent string, discovery *job) error {
	if m.Name == "" {
		return fmt.Errorf("Metric [%s/%d] in %v: Name should not be empty", m.Name, metricIdx, parent)
	}
	if len(m.Statistics) == 0 {
		return fmt.Errorf("Metric [%s/%d] in %v: Statistics should not be empty", m.Name, metricIdx, parent)
	}
	mPeriod := m.Period
	if mPeriod == 0 && discovery != nil {
		if discovery.Period != 0 {
			mPeriod = discovery.Period
		} else {
			mPeriod = 300
		}
	}
	if mPeriod < 1 {
		return fmt.Errorf("Metric [%s/%d] in %v: Period value should be a positive integer", m.Name, metricIdx, parent)
	}
	mLength := m.Length
	if mLength == 0 && discovery != nil {
		if discovery.Length != 0 {
			mLength = discovery.Length
		} else {
			mLength = 120
		}
	}

	mDelay := m.Delay
	if mDelay == 0 && discovery != nil {
		if discovery.Delay != 0 {
			mDelay = discovery.Delay
		} else {
			mDelay = 120
		}
	}

	mNilToZero := m.NilToZero
	if mNilToZero == nil && discovery != nil {
		if discovery.NilToZero != nil {
			mNilToZero = discovery.NilToZero
		} else {
			mNilToZero = aws.Bool(false)
		}
	}

	mAddCloudwatchTimestamp := m.AddCloudwatchTimestamp
	if mAddCloudwatchTimestamp == nil && discovery != nil {
		if discovery.AddCloudwatchTimestamp != nil {
			mAddCloudwatchTimestamp = discovery.AddCloudwatchTimestamp
		} else {
			mAddCloudwatchTimestamp = aws.Bool(false)
		}
	}

	if mLength < mPeriod {
		log.Warningf(
			"Metric [%s/%d] in %v: length(%d) is smaller than period(%d). This can cause that the data requested is not ready and generate data gaps",
			m.Name, metricIdx, parent, mLength, mPeriod)
	}
	m.Length = mLength
	m.Period = mPeriod
	m.Delay = mDelay
	m.NilToZero = mNilToZero
	m.AddCloudwatchTimestamp = mAddCloudwatchTimestamp

	return nil
}
