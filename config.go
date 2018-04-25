package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
)

type conf struct {
	Jobs []job `yaml:"jobs"`
}

type job struct {
	Name          string         `yaml:"name"`
	Region        string         `yaml:"region"`
	Type          string         `yaml:"type"`
	DiscoveryTags []discoveryTag `yaml:"discoveryTags"`
	ExportedTags  []string       `yaml:"exportedTags"`
	Metrics       []metric       `yaml:"metrics"`
}

type metric struct {
	Name       string `yaml:"name"`
	Statistics string `yaml:"statistics"`
	Period     int    `yaml:"period"`
	Length     int    `yaml:"length"`
}

type discoveryTag struct {
	Key   string `yaml:"Key"`
	Value string `yaml:"Value"`
}

func (c *conf) getConf(file *string) *conf {
	yamlFile, err := ioutil.ReadFile(*file)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}
