package main

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type conf struct {
	Jobs []job `yaml:"jobs"`
}

type job struct {
	Discovery discovery `yaml:"discovery"`
	Metrics   []metric  `yaml:"metrics"`
}

type discovery struct {
	Region       string   `yaml:"region"`
	Type         string   `yaml:"type"`
	SearchTags   []tag    `yaml:"searchTags"`
	ExportedTags []string `yaml:"exportedTags"`
}

type metric struct {
	Name       string `yaml:"name"`
	Statistics string `yaml:"statistics"`
	Period     int    `yaml:"period"`
	Length     int    `yaml:"length"`
	NilToZero  bool   `yaml:"nilToZero"`
}

type tag struct {
	Key   string `yaml:"Key"`
	Value string `yaml:"Value"`
}

func (c *conf) loadConf(file *string) error {
	yamlFile, err := ioutil.ReadFile(*file)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		return err
	}

	for _, job := range c.Jobs {
		if !stringInSlice(job.Discovery.Type, supportedServices) {
			return fmt.Errorf("Service is not in known list!: %v", job.Discovery.Type)
		}
	}
	return nil
}
