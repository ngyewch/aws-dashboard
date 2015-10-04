package main

import (
	"io/ioutil"

	"github.com/ghodss/yaml"
)

type General struct {
	DefaultRegion string `json:"defaultRegion"`
}

type Billing struct {
	BucketName string `json:"bucketName"`
}

type Config struct {
	General General `json:"general"`
	Billing Billing `json:"billing"`
}

func readConfig(filename string) (cfg Config, err error) {
	var config Config

	configData, err := ioutil.ReadFile("aws-dashboard.yaml")
	if err != nil {
		return config, err
	}

	err = yaml.Unmarshal(configData, &config)
	if err != nil {
		return config, err
	}

	return config, nil
}
