package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type conf struct {
	Discovery []discovery `yaml:"discovery"`
	Static    []static    `yaml:"static"`
}

type discovery struct {
	Region     string   `yaml:"region"`
	Type       string   `yaml:"type"`
	SearchTags []tag    `yaml:"searchTags"`
	Metrics    []metric `yaml:"metrics"`
}

type static struct {
	Name       string      `yaml:"name"`
	Region     string      `yaml:"region"`
	Namespace  string      `yaml:"namespace"`
	Dimensions []dimension `yaml:"dimensions"`
	Metrics    []metric    `yaml:"metrics"`
}

type metric struct {
	Name       string   `yaml:"name"`
	Statistics []string `yaml:"statistics"`
	Period     int      `yaml:"period"`
	Length     int      `yaml:"length"`
	NilToZero  bool     `yaml:"nilToZero"`
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

	for _, job := range c.Discovery {
		if !stringInSlice(job.Type, supportedServices) {
			return fmt.Errorf("Service is not in known list!: %v", job.Type)
		}
	}
	return nil
}
