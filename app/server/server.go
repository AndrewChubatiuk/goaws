package server

import (
	"net/http"
	log "github.com/sirupsen/logrus"
	"fmt"
	"github.com/p4tin/goaws/services/sns"
	"github.com/p4tin/goaws/services/sqs"
)

type Server struct {
	SQS sqs.Service
	SNS sns.Service
}

func New(config conf.Config) *Server {
	srv = &Server{
		SQS = sqs.New(config),
		SNS = sns.New(config),
	}
	return srv
}

func (srv * Server) Serve() {
	quit := make(chan bool)
	go srv.SQS.PeriodicTasks(1 * time.Second, quit)

	if srv.SQS.Port == srv.SNS.Port {
		if srv.SQS.Listen != srv.SNS.Listen {
			fmt.Errorf("Addresses are different but ports are equal. Please set different ports or use the same listen address for both services")
		}
		http.Handle("/", srv.SQS.Router)
    	http.Handle("/", srv.SNS.Router)

		go serve("SNS and SQS", srv.SQS.Listen, nil)
	} else {
		go serve("SQS", srv.SQS.Listen, srv.SQS.Router)
		go serve("SNS", srv.SNS.Listen, srv.SNS.Router)
	}
	<-quit
}

func serve(component string, listen string, r http.Handler) {
	log.Warnf("GoAws %s listening on: %s", component, listen)
	err := http.ListenAndServe(listen, r)
	log.Fatal(err)
}
