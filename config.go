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
		if !stringInSlice(job.Type, supportedServices) {
			return fmt.Errorf("Service is not in known list!: %v", job.Type)
		}

		for _, metric := range job.Metrics {
			if job.Length < 300 && metric.Length < 300 {
				log.Warn("WATCH OUT! - Metric length of less than 5 minutes configured which is default for most cloudwatch metrics e.g. ELBs")
			}

			if job.Period < 1 && metric.Period < 1 {
				return fmt.Errorf("Period value should be a positive integer")
			}
		}
	}
	for n, job := range c.Static {
		if len(job.RoleArns) == 0 {
			c.Static[n].RoleArns = []string{""} // use current IAM role
		}
	}
	return nil
}
