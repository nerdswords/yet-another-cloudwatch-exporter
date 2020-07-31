package main

import (
	"fmt"
	"io/ioutil"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type conf struct {
	Discovery discovery `yaml:"discovery"`
	Static    []static  `yaml:"static"`
}

type discovery struct {
	ExportedTagsOnMetrics exportedTagsOnMetrics `yaml:"exportedTagsOnMetrics"`
	Jobs                  []job                 `yaml:"jobs"`
}

type exportedTagsOnMetrics map[string][]string

type job struct {
	Regions                []string `yaml:"regions"`
	Type                   string   `yaml:"type"`
	RoleArns               []string `yaml:"roleArns"`
	AwsDimensions          []string `yaml:"awsDimensions"`
	SearchTags             []tag    `yaml:"searchTags"`
	CustomTags             []tag    `yaml:"customTags"`
	Metrics                []metric `yaml:"metrics"`
	Length                 int      `yaml:"length"`
	Delay                  int      `yaml:"delay"`
	Period                 int      `yaml:"period"`
	AddCloudwatchTimestamp bool     `yaml:"addCloudwatchTimestamp"`
}

type static struct {
	Name       string      `yaml:"name"`
	Regions    []string    `yaml:"regions"`
	RoleArns   []string    `yaml:"roleArns"`
	Namespace  string      `yaml:"namespace"`
	CustomTags []tag       `yaml:"customTags"`
	Dimensions []dimension `yaml:"dimensions"`
	Metrics    []metric    `yaml:"metrics"`
}

type metric struct {
	Name                   string      `yaml:"name"`
	Statistics             []string    `yaml:"statistics"`
	AdditionalDimensions   []dimension `yaml:"additionalDimensions"`
	Period                 int         `yaml:"period"`
	Length                 int         `yaml:"length"`
	Delay                  int         `yaml:"delay"`
	NilToZero              bool        `yaml:"nilToZero"`
	AddCloudwatchTimestamp bool        `yaml:"addCloudwatchTimestamp"`
}

type dimension struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type tag struct {
	Key   string `yaml:"Key"`
	Value string `yaml:"Value"`
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
			err := c.validateDiscoveryJob(job, idx)
			if err != nil {
				return err
			}
		}
	}

	if c.Static != nil {
		for idx, job := range c.Static {
			err := c.validateStaticJob(job, idx)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *conf) validateDiscoveryJob(j job, jobIdx int) error {
	if len(j.Regions) == 0 {
		return fmt.Errorf("Discovery job [%v]: Regions should not be empty", jobIdx)
	}
	if j.Type != "" {
		if !stringInSlice(j.Type, supportedServices) {
			return fmt.Errorf("Discovery job [%v]: Service is not in known list!: %v", jobIdx, j.Type)
		}
	} else {
		return fmt.Errorf("Discovery job [%v]: Type should not be empty", jobIdx)
	}
	if len(j.Metrics) == 0 {
		return fmt.Errorf("Discovery job [%v]: Metrics should not be empty", jobIdx)
	}
	for metricIdx, metric := range j.Metrics {
		err := c.validateMetric(metric, metricIdx, fmt.Sprintf("Discovery job [%v]", jobIdx))
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *conf) validateStaticJob(j static, jobIdx int) error {
	if j.Name == "" {
		return fmt.Errorf("Static job [%v]: Name should not be empty", jobIdx)
	}
	if j.Namespace == "" {
		return fmt.Errorf("Static job [%v]: Namespace should not be empty", jobIdx)
	}
	if len(j.Regions) == 0 {
		return fmt.Errorf("Static job [%v]: Regions should not be empty", jobIdx)
	}
	for metricIdx, metric := range j.Metrics {
		err := c.validateMetric(metric, metricIdx, fmt.Sprintf("Static job [%v]", jobIdx))
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *conf) validateMetric(m metric, metricIdx int, parent string) error {
	if m.Name == "" {
		return fmt.Errorf("Metric [%v] in %v: Name should not be empty", metricIdx, parent)
	}
	if len(m.Statistics) == 0 {
		return fmt.Errorf("Metric [%v] in %v: Statistics should not be empty", metricIdx, parent)
	}
	if m.Period < 1 {
		return fmt.Errorf("Metric [%v] in %v: Period value should be a positive integer", metricIdx, parent)
	}
	if m.Length < m.Period {
		log.Warningf("Metric [%v] in %v: length is smaller than period. This can cause that the data requested is not ready and generate data gaps", metricIdx, parent)
	}

	return nil
}
