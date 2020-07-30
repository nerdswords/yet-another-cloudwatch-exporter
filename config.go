package main

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
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

// s3elements parses bucket and key from s3://bucket_name/key_name string
func (c *conf) s3elements(url string) (bucket, key string) {
	fields := strings.SplitN(url, "/", 4)
	if len(fields) < 4 {
		return "", ""
	}
	bucket = fields[2]
	key = fields[3]
	return bucket, key
}

// loadContent from s3 bucket or local file
func (c *conf) loadContent(file string) ([]byte, error) {
	if strings.HasPrefix(file, "s3://") {
		bucket, key := c.s3elements(file)
		sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))
		downloader := s3manager.NewDownloader(sess)
		buf := aws.NewWriteAtBuffer([]byte{})
		nBytes, err := downloader.Download(buf,
			&s3.GetObjectInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(key),
			})
		log.Debugf("Downloaded %d bytes from %s", nBytes, file)
		return buf.Bytes(), err
	}
	return ioutil.ReadFile(file)
}

func (c *conf) load(file string) error {
	yamlFile, err := c.loadContent(file)
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
	if j.Regions != nil {
		if len(j.Regions) == 0 {
			return fmt.Errorf("Discovery job [%v]: Regions should not be empty", jobIdx)
		}
	} else {
		return fmt.Errorf("Discovery job [%v]: Regions should not be empty", jobIdx)
	}
	if j.Type != "" {
		if !stringInSlice(j.Type, supportedServices) {
			return fmt.Errorf("Discovery job [%v]: Service is not in known list!: %v", jobIdx, j.Type)
		}
	} else {
		return fmt.Errorf("Discovery job [%v]: Type should not be empty", jobIdx)
	}
	if j.Metrics != nil {
		if len(j.Metrics) == 0 {
			return fmt.Errorf("Discovery job [%v]: Metrics should not be empty", jobIdx)
		}
	} else {
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
	if j.Regions != nil {
		if len(j.Regions) == 0 {
			return fmt.Errorf("Static job [%v]: Regions should not be empty", jobIdx)
		}
	} else {
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
	if m.Statistics != nil {
		if len(m.Statistics) == 0 {
			return fmt.Errorf("Metric [%v] in %v: Statistics should not be empty", metricIdx, parent)
		}
	} else {
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
