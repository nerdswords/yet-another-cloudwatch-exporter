package exporter

import (
	"fmt"
	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type ScrapeConf struct {
	Discovery Discovery `yaml:"discovery"`
	Static    []*Static `yaml:"static"`
}

type Discovery struct {
	ExportedTagsOnMetrics exportedTagsOnMetrics `yaml:"exportedTagsOnMetrics"`
	Jobs                  []*Job                `yaml:"jobs"`
}

type exportedTagsOnMetrics map[string][]string

type Job struct {
	Regions                []string  `yaml:"regions"`
	Type                   string    `yaml:"type"`
	RoleArns               []string  `yaml:"roleArns"`
	SearchTags             []Tag     `yaml:"searchTags"`
	CustomTags             []Tag     `yaml:"customTags"`
	Metrics                []*Metric `yaml:"metrics"`
	Length                 int       `yaml:"length"`
	Delay                  int       `yaml:"delay"`
	Period                 int       `yaml:"period"`
	AddCloudwatchTimestamp *bool     `yaml:"addCloudwatchTimestamp"`
	NilToZero              *bool     `yaml:"nilToZero"`
}

type Static struct {
	Name       string      `yaml:"name"`
	Regions    []string    `yaml:"regions"`
	RoleArns   []string    `yaml:"roleArns"`
	Namespace  string      `yaml:"namespace"`
	CustomTags []Tag       `yaml:"customTags"`
	Dimensions []Dimension `yaml:"dimensions"`
	Metrics    []*Metric   `yaml:"metrics"`
}

type Metric struct {
	Name                   string   `yaml:"name"`
	Statistics             []string `yaml:"statistics"`
	Period                 int      `yaml:"period"`
	Length                 int      `yaml:"length"`
	Delay                  int      `yaml:"delay"`
	NilToZero              *bool    `yaml:"nilToZero"`
	AddCloudwatchTimestamp *bool    `yaml:"addCloudwatchTimestamp"`
}

type Dimension struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type Tag struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

func (c *ScrapeConf) Load(file *string) error {
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

func (c *ScrapeConf) validate() error {
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

func (j *Job) validateDiscoveryJob(jobIdx int) error {
	if j.Type != "" {
		if SupportedServices.GetService(j.Type) == nil {
			return fmt.Errorf("Discovery job [%d]: Service is not in known list!: %s", jobIdx, j.Type)
		}
	} else {
		return fmt.Errorf("Discovery job [%d]: Type should not be empty", jobIdx)
	}
	if len(j.Regions) == 0 {
		return fmt.Errorf("Discovery job [%s/%d]: Regions should not be empty", j.Type, jobIdx)
	}
	if len(j.Metrics) == 0 {
		return fmt.Errorf("Discovery job [%s/%d]: Metrics should not be empty", j.Type, jobIdx)
	}
	for metricIdx, metric := range j.Metrics {
		parent := fmt.Sprintf("Discovery job [%s/%d]", j.Type, jobIdx)
		err := metric.validateMetric(metricIdx, parent, j)
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

func (m *Metric) validateMetric(metricIdx int, parent string, discovery *Job) error {
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
