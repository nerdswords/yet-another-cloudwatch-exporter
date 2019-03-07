package main

import (
	"fmt"
	"io/ioutil"

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
	Region     string   `yaml:"region"`
	Type       string   `yaml:"type"`
	RoleArn    string   `yaml:"roleArn"`
	SearchTags []tag    `yaml:"searchTags"`
	Metrics    []metric `yaml:"metrics"`
}

type static struct {
	Name       string      `yaml:"name"`
	Region     string      `yaml:"region"`
	RoleArn    string      `yaml:"roleArn"`
	Namespace  string      `yaml:"namespace"`
	CustomTags []tag       `yaml:"customTags"`
	Dimensions []dimension `yaml:"dimensions"`
	Metrics    []metric    `yaml:"metrics"`
}

type metric struct {
	Name                  string      `yaml:"name"`
	Statistics            []string    `yaml:"statistics"`
	AdditionalDimensions	[]dimension `yaml:"additionalDimensions"`
	Period                int         `yaml:"period"`
	Length                int         `yaml:"length"`
	Delay                 int         `yaml:"delay"`
	NilToZero             bool        `yaml:"nilToZero"`
	DisableTimestamp      bool        `yaml:"disableTimestamp"`
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

	for _, job := range c.Discovery.Jobs {
		if !stringInSlice(job.Type, supportedServices) {
			return fmt.Errorf("Service is not in known list!: %v", job.Type)
		}
	}
	return nil
}
