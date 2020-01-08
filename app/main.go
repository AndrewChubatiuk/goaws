package main

import (
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/p4tin/goaws/app"

	log "github.com/sirupsen/logrus"

	"github.com/p4tin/goaws/app/conf"
	"github.com/p4tin/goaws/app/gosqs"
	"github.com/p4tin/goaws/app/router"
)

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
	  return false
	}
	return !info.IsDir()
}

func main() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)

	var configFile string
	var environment string
	var config conf.Config{
		LogLevel: "warn",
		ListenAddr: ":4100",
		Region: "local",
		AccountID: "queue",
		QueueAttributeDefaults = EnvQueueAttributes{
			VisibilityTimeout: 30,
			ReceiveMessageWaitTimeSeconds: 0,
		},
	}

	flag.StringVar(&configFile, "config", "./conf/goaws.yaml", "config file location + name")
	flag.StringVar(&environment, "env", "Local", "environment name")
	flag.Parse()

	if fileExists(configFile) {
		config.LoadConfig(configFile, environment)
	}

	if level, err := ParseLevel(config.LogLevel); err != nil {
		log.Fatal(err)
	} else {
		log.SetLevel(level)
		if config.LogFile != "" && fileExists(config.LogFile) {
			if logFile, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err != nil {
				log.Infof("Failed to log to file: %s, using default stderr", config.LogFile)
			} else {
				log.SetOutput(logFile)
			}
		}
	}

	srv := server.New(config)
	srv.Serve()
}
