package router

import (
	"io"
	"net/http"

	log "github.com/sirupsen/logrus"

	"fmt"

	"github.com/gorilla/mux"
	sns "github.com/p4tin/goaws/app/gosns"
	sqs "github.com/p4tin/goaws/app/gosqs"
)

// New returns a new router
func New() http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/", actionHandler).Methods("GET", "POST")
	r.HandleFunc("/health", health).Methods("GET")
	r.HandleFunc("/{account}", actionHandler).Methods("GET", "POST")
	r.HandleFunc("/queue/{queueName}", actionHandler).Methods("GET", "POST")
	r.HandleFunc("/{account}/{queueName}", actionHandler).Methods("GET", "POST")
	r.HandleFunc("/SimpleNotificationService/{id}.pem", pemHandler).Methods("GET")

	return r
}

var routingTable = map[string]http.HandlerFunc{
	// SQS
	"ListQueues":              sqs.ListQueues,
	"CreateQueue":             sqs.CreateQueue,
	"GetQueueAttributes":      sqs.GetQueueAttributes,
	"SetQueueAttributes":      sqs.SetQueueAttributes,
	"SendMessage":             sqs.SendMessage,
	"SendMessageBatch":        sqs.SendMessageBatch,
	"ReceiveMessage":          sqs.ReceiveMessage,
	"DeleteMessage":           sqs.DeleteMessage,
	"DeleteMessageBatch":      sqs.DeleteMessageBatch,
	"GetQueueUrl":             sqs.GetQueueUrl,
	"PurgeQueue":              sqs.PurgeQueue,
	"DeleteQueue":             sqs.DeleteQueue,
	"ChangeMessageVisibility": sqs.ChangeMessageVisibility,

	// SNS
	"ListTopics":                sns.ListTopics,
	"CreateTopic":               sns.CreateTopic,
	"DeleteTopic":               sns.DeleteTopic,
	"Subscribe":                 sns.Subscribe,
	"ConfirmSubscription":       sns.ConfirmSubscription,
	"SetSubscriptionAttributes": sns.SetSubscriptionAttributes,
	"GetSubscriptionAttributes": sns.GetSubscriptionAttributes,
	"ListSubscriptionsByTopic":  sns.ListSubscriptionsByTopic,
	"ListSubscriptions":         sns.ListSubscriptions,
	"Unsubscribe":               sns.Unsubscribe,
	"Publish":                   sns.Publish,
}

func health(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(200)
	fmt.Fprint(w, "OK")
}

func actionHandler(w http.ResponseWriter, req *http.Request) {
	log.WithFields(
		log.Fields{
			"action": req.FormValue("Action"),
			"url":    req.URL,
		}).Debug("Handling URL request")
	fn, ok := routingTable[req.FormValue("Action")]
	if !ok {
		log.Println("Bad Request - Action:", req.FormValue("Action"))
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, "Bad Request")
		return
	}

	http.HandlerFunc(fn).ServeHTTP(w, req)
}

func pemHandler(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write(sns.PemKEY)
}








quit := make(chan bool)

	go gosqs.PeriodicTasks(1 * time.Second, quit)

	if config.ListenSQS == config.ListenSNS {
		serve("SNS and SQS", config.ListenSQS, r)
	} else {
		serve("SQS", config.ListenSQS, r)
		serve("SNS", config.ListenSNS, r)
	}
	<-quit
}

func serve(component string, listen string, r http.Handler) {
	log.Warnf("GoAws %s listening on: %s", component, listen)
	err := http.ListenAndServe(listen, r)
	log.Fatal(err)












	app.SyncQueues.Lock()
	app.SyncTopics.Lock()
	for _, queue := range config.Queues {
		queueUrl = fmt.Sprintf("http://%s.%s/%s/%s", config.Region, config.ListenSQSAddr, config.AccountID, queue.Name)
		queueArn := fmt.Sprintf("arn:aws:sqs:%s:%s:%s", config.Region, config.AccountID, queue.Name)
		app.SyncQueues.Queues[queue.Name] = &app.Queue{
			Name:                queue.Name,
			TimeoutSecs:         app.CurrentEnvironment.QueueAttributeDefaults.VisibilityTimeout,
			Arn:                 queueArn,
			URL:                 queueUrl,
			ReceiveWaitTimeSecs: queue.ReceiveMessageWaitTimeSeconds,
			IsFIFO:              app.HasFIFOQueueName(queue.Name),
		}
	}

	for _, topic := range envs[env].Topics {
		topicArn := "arn:aws:sns:" + app.CurrentEnvironment.Region + ":" + app.CurrentEnvironment.AccountID + ":" + topic.Name

		newTopic := &app.Topic{Name: topic.Name, Arn: topicArn}
		newTopic.Subscriptions = make([]*app.Subscription, 0, 0)

		for _, subs := range topic.Subscriptions {
			var newSub *app.Subscription
			if strings.Contains(subs.Protocol, "http") {
				newSub = createHttpSubscription(subs)
			} else {
				//Queue does not exist yet, create it.
				newSub = createSqsSubscription(subs, topicArn)
			}
			if subs.FilterPolicy != "" {
				filterPolicy := &app.FilterPolicy{}
				err = json.Unmarshal([]byte(subs.FilterPolicy), filterPolicy)
				if err != nil {
					log.Errorf("err: %s", err)
					return ports
				}
				newSub.FilterPolicy = filterPolicy
			}

			newTopic.Subscriptions = append(newTopic.Subscriptions, newSub)
		}
		app.SyncTopics.Topics[topic.Name] = newTopic
	}

	app.SyncQueues.Unlock()
	app.SyncTopics.Unlock()

	return ports
}

func createHttpSubscription(configSubscription app.EnvSubsciption) *app.Subscription {
	newSub := &app.Subscription{EndPoint: configSubscription.EndPoint, Protocol: configSubscription.Protocol, TopicArn: configSubscription.TopicArn, Raw: configSubscription.Raw}
	subArn, _ := common.NewUUID()
	subArn = configSubscription.TopicArn + ":" + subArn
	newSub.SubscriptionArn = subArn
	return newSub
}

func createSqsSubscription(configSubscription app.EnvSubsciption, topicArn string) *app.Subscription {
	if _, ok := app.SyncQueues.Queues[configSubscription.QueueName]; !ok {
		queueUrl := "http://%s/" + app.CurrentEnvironment.AccountID + "/" + configSubscription.QueueName
		if app.CurrentEnvironment.Region != "" {
			queueUrl = "http://" + app.CurrentEnvironment.Region + ".%s/" + app.CurrentEnvironment.AccountID + "/" + configSubscription.QueueName
		}
		queueArn := "arn:aws:sqs:" + app.CurrentEnvironment.Region + ":" + app.CurrentEnvironment.AccountID + ":" + configSubscription.QueueName
		app.SyncQueues.Queues[configSubscription.QueueName] = &app.Queue{
			Name:                configSubscription.QueueName,
			TimeoutSecs:         app.CurrentEnvironment.QueueAttributeDefaults.VisibilityTimeout,
			Arn:                 queueArn,
			URL:                 queueUrl,
			ReceiveWaitTimeSecs: app.CurrentEnvironment.QueueAttributeDefaults.ReceiveMessageWaitTimeSeconds,
			IsFIFO:              app.HasFIFOQueueName(configSubscription.QueueName),
		}
	}
	qArn := app.SyncQueues.Queues[configSubscription.QueueName].Arn
	newSub := &app.Subscription{EndPoint: qArn, Protocol: "sqs", TopicArn: topicArn, Raw: configSubscription.Raw}
	subArn, _ := common.NewUUID()
	subArn = topicArn + ":" + subArn
	newSub.SubscriptionArn = subArn
	return newSub
}
