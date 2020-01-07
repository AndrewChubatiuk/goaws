package conf

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	log "github.com/sirupsen/logrus"

	"gopkg.in/yaml.v2"
	"github.com/p4tin/goaws/app"
	"github.com/p4tin/goaws/app/common"
)

type Subsciption struct {
	Protocol     string
	EndPoint     string
	TopicArn     string
	QueueName    string
	Raw          bool
	FilterPolicy string
}

type Topic struct {
	Name          string
	Subscriptions []Subsciption
}

type Queue struct {
	Name                          string
	QueueAttributes               QueueAttributes
}

type QueueAttributes struct {
	VisibilityTimeout             int
	ReceiveMessageWaitTimeSeconds int
}

type Config struct {
	ListenAddr             string
	ListenSQSAddr          string
	ListenSNSAddr          string
	Region                 string
	AccountID              string
	LogFile                string
	LogLevel               string
	Topics                 []Topic
	Queues                 []Queue
	QueueAttributeDefaults QueueAttributes
}

func (c * Config) LoadConfig(configPath string, environment string) error {
	var fullConfig map[string]interface{}
	log.Warnf("Loading config file: %s", configPath)
	configFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(configFile, &fullConfig)
	if err != nil {
		log.Errorf("err: %v\n", err)
		return nil, err
	}
	if confData, ok := fullConfig[environment]; ok {
		err = c.UnmarshalEnvironment(confData)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("Environment %s is not defined in config", environment)
	}

	if c.ListenSQSAddr == "" {
		c.ListenSQSAddr = config.ListenAddr
	}

	if c.ListenSNSAddr == "" {
		c.ListenSNSAddr = config.ListenAddr
	}
}

func (c * Config) UnmarshalEnvironment(data interface{}) error {
	env, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(env, c)
}
