package gosqs

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gorilla/mux"
	"github.com/p4tin/goaws/app"
	"github.com/p4tin/goaws/app/common"
)

func init() {
	app.SqsErrors = map[string]app.SqsErrorType{
		"QueueNotFound": app.SqsErrorType{
			HttpError: http.StatusBadRequest,
			Type: "Not Found",
			Code: "AWS.SimpleQueueService.NonExistentQueue",
			Message: "The specified queue does not exist for this wsdl version."
		},
		"QueueExists": app.SqsErrorType{
			HttpError: http.StatusBadRequest,
			Type: "Duplicate",
			Code: "AWS.SimpleQueueService.QueueExists",
			Message: "The specified queue already exists."
		},
		"MessageDoesNotExist": app.SqsErrorType{
			HttpError: http.StatusNotFound,
			Type: "Not Found",
			Code: "AWS.SimpleQueueService.QueueExists",
			Message: "The specified queue does not contain the message specified."
		},
		"GeneralError": app.SqsErrorType{
			HttpError: http.StatusBadRequest,
			Type: "GeneralError",
			Code: "AWS.SimpleQueueService.GeneralError",
			Message: "General Error."
		},
		"TooManyEntriesInBatchRequest": app.SqsErrorType{
			HttpError: http.StatusBadRequest,
			Type: "TooManyEntriesInBatchRequest",
			Code: "AWS.SimpleQueueService.TooManyEntriesInBatchRequest",
			Message: "Maximum number of entries per request are 10."
		},
		"BatchEntryIdsNotDistinct": app.SqsErrorType{
			HttpError: http.StatusBadRequest,
			Type: "BatchEntryIdsNotDistinct",
			Code: "AWS.SimpleQueueService.BatchEntryIdsNotDistinct",
			Message: "Two or more batch entries in the request have the same Id."
		},
		"EmptyBatchRequest": app.SqsErrorType{
			HttpError: http.StatusBadRequest,
			Type: "EmptyBatchRequest",
			Code: "AWS.SimpleQueueService.EmptyBatchRequest",
			Message: "The batch request doesn't contain any entries."
		},
		"InvalidVisibilityTimeout": app.SqsErrorType{
			HttpError: http.StatusBadRequest,
			Type: "ValidationError",
			Code: "AWS.SimpleQueueService.ValidationError",
			Message: "The visibility timeout is incorrect"
		},
		"MessageNotInFlight": app.SqsErrorType{
			HttpError: http.StatusBadRequest,
			Type: "MessageNotInFlight",
			Code: "AWS.SimpleQueueService.MessageNotInFlight",
			Message: "The message referred to isn't in flight."
		},
		"InvalidParameterValue": app.SqsErrorType{
			HttpError: http.StatusBadRequest,
			Type:      "InvalidParameterValue",
			Code:      "AWS.SimpleQueueService.InvalidParameterValue",
			Message:   "An invalid or out-of-range value was supplied for the input parameter.",
		},
		"InvalidAttributeValue": app.SqsErrorType{
			HttpError: http.StatusBadRequest,
			Type:      "InvalidAttributeValue",
			Code:      "AWS.SimpleQueueService.InvalidAttributeValue",
			Message:   "Invalid Value for the parameter RedrivePolicy.",
		}
	}
}

func (sqs * SQS) PeriodicTasks(d time.Duration, quit <-chan struct{}) {
	ticker := time.NewTicker(d)
	for {
		select {
		case <-ticker.C:
			sqs.Lock()
			for j := range sqs.Queues {
				queue := sqs.Queues[j]

				log.Debugf("Queue [%s] length [%d]", queue.Name, len(queue.Messages))
				for i := 0; i < len(queue.Messages); i++ {
					msg := &queue.Messages[i]
					if msg.ReceiptHandle != "" {
						if msg.VisibilityTimeout.Before(time.Now()) {
							log.Debugf("Making message visible again %s", msg.ReceiptHandle)
							queue.UnlockGroup(msg.GroupID)
							msg.ReceiptHandle = ""
							msg.ReceiptTime = time.Now().UTC()
							msg.Retry++
							if queue.MaxReceiveCount > 0 &&
								queue.DeadLetterQueue != nil &&
								msg.Retry > queue.MaxReceiveCount {
								queue.DeadLetterQueue.Messages = append(queue.DeadLetterQueue.Messages, *msg)
								queue.Messages = append(queue.Messages[:i], queue.Messages[i+1:]...)
								i++
							}
						}
					}
				}
			}
			sqs.Unlock()
		case <-quit:
			ticker.Stop()
			return
		}
	}
}

func (sqs * SQS) ListQueues(args map[string]string) (*ListQueuesResponse, error) {
	resp := &ListQueuesResponse{
		Response: &NewResponse()
		Result: &ListQueuesResult{
			QueueUrl = make([]string, 0)
		}
	}
	queueNamePrefix := args["QueueNamePrefix"]
	host := args["Host"]
	log.Println("Listing Queues")
	for _, queue := range sqs.Queues {
		sqs.Lock()
		if strings.HasPrefix(queue.Name, queueNamePrefix) {
			resp.Result.QueueUrl = append(resp.Result.QueueUrl, sqs.GetQueueUrl(queue, host))
		}
		sqs.Unlock()
	}
	return resp
}

func (sqs * SQS) CreateQueue(args map[string]string) (*cCreateQueueResponse, error) {
	queueName := args["QueueName"]
	queueUrl := sqs.QueueUrl(queueName, host)
	if _, ok := sqs.Queues[queueName]; !ok {
		log.Println("Creating Queue:", queueName)
		queue := &app.Queue{
			Name:                queueName,
			URL:                 sqs.QueueUrl(queueName, host),
			Arn:                 sqs.QueueArn(queueName),
			TimeoutSecs:         sqs.QueueAttributeDefaults.VisibilityTimeout,
			ReceiveWaitTimeSecs: sqs.QueueAttributeDefaults.ReceiveMessageWaitTimeSeconds,
			IsFIFO:              app.HasFIFOQueueName(queueName),
		}
		if err := validateAndSetQueueAttributes(queue, req.Form); err != nil {
			createErrorResponse(err.Error())
			return
		}
		s.Lock()
		s.Queues[queueName] = queue
		s.Unlock()
	}

	return &CreateQueueResponse{
		Response: &NewResponse(),
		Result: &CreateQueueResult{
			QueueUrl: queueUrl,
		},
	}
}

func (sqs * SQS) SendMessage(args map[string]string) (*SendMessageResponse, error){
	vars := mux.Vars(req)
	queueName := vars["queueName"]
	messageBody := args["MessageBody"]
	messageGroupID := args["MessageGroupId"]
	messageAttributes := extractMessageAttributes(req, "")
	if _, ok := sqs.Queues[queueName]; !ok {
		NewErrorResponse("QueueNotFound")
		return
	}
	log.Println("Putting Message in Queue:", queueName)
	msg := &Message{
		MessageBody: []byte(messageBody),
		MD5OfMessageBody: common.GetMD5Hash(messageBody),
		Uuid: common.NewUUID(),
		GroupID: messageGroupID,
	}
	if len(messageAttributes) > 0 {
		msg.MessageAttributes = messageAttributes
		msg.MD5OfMessageAttributes = common.HashAttributes(messageAttributes)
	}
	sqs.Lock()
	fifoSeqNumber := ""
	if sqs.Queues[queueName].IsFIFO {
		fifoSeqNumber = sqs.Queues[queueName].NextSequenceNumber(messageGroupID)
	}
	sqs.Queues[queueName].Messages = append(sqs.Queues[queueName].Messages, msg)
	sqs.Unlock()
	log.Infof("%s: Queue: %s, Message: %s\n", time.Now().Format("2006-01-02 15:04:05"), queueName, msg.MessageBody)

	return &SendMessageResponse{
		Response: &NewResponse(),
		Result: &SendMessageResult{
			MD5OfMessageAttributes: msg.MD5OfMessageAttributes,
			MD5OfMessageBody:       msg.MD5OfMessageBody,
			MessageId:              msg.Uuid,
			SequenceNumber:         fifoSeqNumber,
		},
	}, nil
}

type SendEntry struct {
	Id                     string
	MessageBody            string
	MessageAttributes      map[string]app.MessageAttributeValue
	MessageGroupId         string
	MessageDeduplicationId string
}

func (sqs * SQS) SendMessageBatch(args map[string]string) (*SendMessageBatchResponse, error) {
	vars := mux.Vars(req)
	queueName := vars["queueName"]
	if _, ok := sqs.Queues[queueName]; !ok {
		createErrorResponse("QueueNotFound")
		return
	}
	sendEntries := []SendEntry{}
	for k, v := range req.Form {
		keySegments := strings.Split(k, ".")
		if keySegments[0] == "SendMessageBatchRequestEntry" {
			if len(keySegments) < 3 {
				createErrorResponse(w, req, "EmptyBatchRequest")
				return
			}
			keyIndex, err := strconv.Atoi(keySegments[1])

			if err != nil {
				createErrorResponse("Error")
				return
			}

			if len(sendEntries) < keyIndex {
				newSendEntries := make([]SendEntry, keyIndex)
				copy(newSendEntries, sendEntries)
				sendEntries = newSendEntries
			}

			if keySegments[2] == "Id" {
				sendEntries[keyIndex-1].Id = v[0]
			}

			if keySegments[2] == "MessageBody" {
				sendEntries[keyIndex-1].MessageBody = v[0]
			}

			if keySegments[2] == "MessageGroupId" {
				sendEntries[keyIndex-1].MessageGroupId = v[0]
			}

			if keySegments[2] == "MessageAttribute" {
				sendEntries[keyIndex-1].MessageAttributes = extractMessageAttributes(req, strings.Join(keySegments[0:2], "."))
			}
		}
	}

	if len(sendEntries) == 0 {
		createErrorResponse("EmptyBatchRequest")
		return
	}

	if len(sendEntries) > 10 {
		createErrorResponse("TooManyEntriesInBatchRequest")
		return
	}
	ids := map[string]struct{}{}
	for _, v := range sendEntries {
		if _, ok := ids[v.Id]; ok {
			createErrorResponse(w, req, "BatchEntryIdsNotDistinct")
			return
		}
		ids[v.Id] = struct{}{}
	}

	sentEntries := make([]app.SendMessageBatchResultEntry, 0)
	log.Println("Putting Message in Queue:", queueName)
	for _, sendEntry := range sendEntries {
		msg := app.Message{MessageBody: []byte(sendEntry.MessageBody)}
		if len(sendEntry.MessageAttributes) > 0 {
			msg.MessageAttributes = sendEntry.MessageAttributes
			msg.MD5OfMessageAttributes = common.HashAttributes(sendEntry.MessageAttributes)
		}
		msg.MD5OfMessageBody = common.GetMD5Hash(sendEntry.MessageBody)
		msg.GroupID = sendEntry.MessageGroupId
		msg.Uuid, _ = common.NewUUID()
		sqs.Lock()
		fifoSeqNumber := ""
		if sqs.Queues[queueName].IsFIFO {
			fifoSeqNumber = sqs.Queues[queueName].NextSequenceNumber(sendEntry.MessageGroupId)
		}
		sqs.Queues[queueName].Messages = append(sqs.Queues[queueName].Messages, msg)
		sqs..Unlock()
		se := app.SendMessageBatchResultEntry{
			Id:                     sendEntry.Id,
			MessageId:              msg.Uuid,
			MD5OfMessageBody:       msg.MD5OfMessageBody,
			MD5OfMessageAttributes: msg.MD5OfMessageAttributes,
			SequenceNumber:         fifoSeqNumber,
		}
		sentEntries = append(sentEntries, se)
		log.Infof("%s: Queue: %s, Message: %s\n", time.Now().Format("2006-01-02 15:04:05"), queueName, msg.MessageBody)
	}

	return &SendMessageBatchResponse{
		Response: &NewResponse(),
		Result: &SendMessageBatchResult{
			Entry: sentEntries
		},
	}, nil
}

func (sqs * SQS) ReceiveMessage(args map[string]string) (*ReceiveMessageResponse, error) {
	waitTimeSeconds := 0
	wts := args["WaitTimeSeconds"]
	if wts != "" {
		waitTimeSeconds, _ = strconv.Atoi(wts)
	}
	maxNumberOfMessages := 1
	mom := args["MaxNumberOfMessages"]
	if mom != "" {
		maxNumberOfMessages, _ = strconv.Atoi(mom)
	}

	vars := mux.Vars(req)
	queueName := vars["queueName"]

	if _, ok := sqs.Queues[queueName]; !ok {
		createErrorResponse(w, req, "QueueNotFound")
		return
	}

	var messages []*app.ResultMessage
	//	respMsg := ResultMessage{}
	respStruct := app.ReceiveMessageResponse{}

	if waitTimeSeconds == 0 {
		sqs.RLock()
		waitTimeSeconds = sqs.Queues[queueName].ReceiveWaitTimeSecs
		sqs.RUnlock()
	}

	loops := waitTimeSeconds * 10
	for loops > 0 {
		sqs.RLock()
		_, queueFound := sqs.Queues[queueName]
		if !queueFound {
			sqs.RUnlock()
			createErrorResponse(w, req, "QueueNotFound")
			return
		}
		messageFound := len(sqs.Queues[queueName].Messages)-numberOfHiddenMessagesInQueue(*sqs.Queues[queueName]) != 0
		sqs.RUnlock()
		if !messageFound {
			continueTimer := time.NewTimer(100 * time.Millisecond)
			select {
			case <-req.Context().Done():
				continueTimer.Stop()
				return // client gave up
			case <-continueTimer.C:
				continueTimer.Stop()
			}
			loops--
		} else {
			break
		}

	}
	log.Println("Getting Message from Queue:", queueName)

	sqs.Lock() // Lock the Queues
	if len(sqs.Queues[queueName].Messages) > 0 {
		numMsg := 0
		messages = make([]*app.ResultMessage, 0)
		for i := range sqs.Queues[queueName].Messages {
			if numMsg >= maxNumberOfMessages {
				break
			}

			if sqs.Queues[queueName].Messages[i].ReceiptHandle != "" {
				continue
			}

			uuid, _ := common.NewUUID()

			msg := &sqs.Queues[queueName].Messages[i]
			msg.ReceiptHandle = msg.Uuid + "#" + uuid
			msg.ReceiptTime = time.Now().UTC()
			msg.VisibilityTimeout = time.Now().Add(time.Duration(sqs.Queues[queueName].TimeoutSecs) * time.Second)

			if sqs.Queues[queueName].IsFIFO {
				// If we got messages here it means we have not processed it yet, so get next
				if sqs.Queues[queueName].IsLocked(msg.GroupID) {
					continue
				}
				// Otherwise lock messages for group ID
				sqs.Queues[queueName].LockGroup(msg.GroupID)
			}

			messages = append(messages, getMessageResult(msg))

			numMsg++
		}

		//		respMsg = ResultMessage{MessageId: messages.Uuid, ReceiptHandle: messages.ReceiptHandle, MD5OfBody: messages.MD5OfMessageBody, Body: messages.MessageBody, MD5OfMessageAttributes: messages.MD5OfMessageAttributes}
		respStruct = app.ReceiveMessageResponse{
			"http://queue.amazonaws.com/doc/2012-11-05/",
			app.ReceiveMessageResult{
				Message: messages,
			},
			app.ResponseMetadata{
				RequestId: "00000000-0000-0000-0000-000000000000",
			},
		}
	} else {
		log.Println("No messages in Queue:", queueName)
		respStruct = app.ReceiveMessageResponse{Xmlns: "http://queue.amazonaws.com/doc/2012-11-05/", Result: app.ReceiveMessageResult{}, Metadata: app.ResponseMetadata{RequestId: "00000000-0000-0000-0000-000000000000"}}
	}
	sqs.Unlock() // Unlock the Queues
}

func numberOfHiddenMessagesInQueue(queue app.Queue) int {
	num := 0
	for i := range queue.Messages {
		if queue.Messages[i].ReceiptHandle != "" {
			num++
		}
	}
	return num
}

func (sqs * SQS) ChangeMessageVisibility(args map[string]string) (*ChangeMessageVisibilityResponse, error) {
	vars := mux.Vars(req)
	queueName := vars["queueName"]
	receiptHandle := args["ReceiptHandle"]
	visibilityTimeout, err := strconv.Atoi(req.FormValue("VisibilityTimeout"))
	if err != nil {
		createErrorResponse("ValidationError")
		return
	}
	if visibilityTimeout > 43200 {
		createErrorResponse("ValidationError")
		return
	}

	if _, ok := sqs.Queues[queueName]; !ok {
		createErrorResponse("QueueNotFound")
		return
	}

	sqs.Lock()
	messageFound := false
	for i := 0; i < len(sqs.Queues[queueName].Messages); i++ {
		queue := sqs.Queues[queueName]
		msgs := queue.Messages
		if msgs[i].ReceiptHandle == receiptHandle {
			timeout := sqs.Queues[queueName].TimeoutSecs
			if visibilityTimeout == 0 {
				msgs[i].ReceiptTime = time.Now().UTC()
				msgs[i].ReceiptHandle = ""
				msgs[i].VisibilityTimeout = time.Now().Add(time.Duration(timeout) * time.Second)
				msgs[i].Retry++
				if queue.MaxReceiveCount > 0 &&
					queue.DeadLetterQueue != nil &&
					msgs[i].Retry > queue.MaxReceiveCount {
					queue.DeadLetterQueue.Messages = append(queue.DeadLetterQueue.Messages, msgs[i])
					queue.Messages = append(queue.Messages[:i], queue.Messages[i+1:]...)
					i++
				}
			} else {
				msgs[i].VisibilityTimeout = time.Now().Add(time.Duration(visibilityTimeout) * time.Second)
			}
			messageFound = true
			break
		}
	}
	sqs.Unlock()
	if !messageFound {
		createErrorResponse("MessageNotInFlight")
		return
	}

	return &app.ChangeMessageVisibilityResult{
		"http://queue.amazonaws.com/doc/2012-11-05/",
		app.ResponseMetadata{RequestId: "00000000-0000-0000-0000-000000000001"}}
}

type DeleteEntry struct {
	Id            string
	ReceiptHandle string
	Error         string
	Deleted       bool
}

func (sqs * SQS) DeleteMessageBatch(args map[string]string) (*DeleteMessageBatchResponse, error) {
	vars := mux.Vars(req)
	queueName := vars["queueName"]
	deleteEntries := []DeleteEntry{}

	for k, v := range req.Form {
		keySegments := strings.Split(k, ".")
		if keySegments[0] == "DeleteMessageBatchRequestEntry" {
			keyIndex, err := strconv.Atoi(keySegments[1])

			if err != nil {
				createErrorResponse(w, req, "Error")
				return
			}

			if len(deleteEntries) < keyIndex {
				newDeleteEntries := make([]DeleteEntry, keyIndex)
				copy(newDeleteEntries, deleteEntries)
				deleteEntries = newDeleteEntries
			}

			if keySegments[2] == "Id" {
				deleteEntries[keyIndex-1].Id = v[0]
			}

			if keySegments[2] == "ReceiptHandle" {
				deleteEntries[keyIndex-1].ReceiptHandle = v[0]
			}
		}
	}

	deletedEntries := make([]app.DeleteMessageBatchResultEntry, 0)

	sqs.Lock()
	if _, ok := sqs.Queues[queueName]; ok {
		for _, deleteEntry := range deleteEntries {
			for i, msg := range sqs.Queues[queueName].Messages {
				if msg.ReceiptHandle == deleteEntry.ReceiptHandle {
					// Unlock messages for the group
					log.Printf("FIFO Queue %s unlocking group %s:", queueName, msg.GroupID)
					sqs.Queues[queueName].UnlockGroup(msg.GroupID)
					sqs.Queues[queueName].Messages = append(sqs.Queues[queueName].Messages[:i], sqs.Queues[queueName].Messages[i+1:]...)

					deleteEntry.Deleted = true
					deletedEntry := app.DeleteMessageBatchResultEntry{Id: deleteEntry.Id}
					deletedEntries = append(deletedEntries, deletedEntry)
					break
				}
			}
		}
	}
	sqs.Unlock()

	notFoundEntries := make([]app.BatchResultErrorEntry, 0)
	for _, deleteEntry := range deleteEntries {
		if deleteEntry.Deleted {
			notFoundEntries = append(notFoundEntries, app.BatchResultErrorEntry{
				Code:        "1",
				Id:          deleteEntry.Id,
				Message:     "Message not found",
				SenderFault: true})
		}
	}

	respStruct := app.DeleteMessageBatchResponse{
		"http://queue.amazonaws.com/doc/2012-11-05/",
		app.DeleteMessageBatchResult{Entry: deletedEntries, Error: notFoundEntries},
		app.ResponseMetadata{RequestId: "00000000-0000-0000-0000-000000000001"}}
}

func (sqs * SQS) DeleteMessage(args map[string]string) (*DeleteMessageResponse, error) {
	receiptHandle := req.FormValue("ReceiptHandle")

	// Retrieve FormValues required
	vars := mux.Vars(req)
	queueName := vars["queueName"]

	log.Println("Deleting Message, Queue:", queueName, ", ReceiptHandle:", receiptHandle)

	// Find queue/message with the receipt handle and delete
	sqs.Lock()
	if _, ok := sqs.Queues[queueName]; ok {
		for i, msg := range sqs.Queues[queueName].Messages {
			if msg.ReceiptHandle == receiptHandle {
				// Unlock messages for the group
				log.Printf("FIFO Queue %s unlocking group %s:", queueName, msg.GroupID)
				sqs.Queues[queueName].UnlockGroup(msg.GroupID)
				//Delete message from Q
				sqs.Queues[queueName].Messages = append(sqs.Queues[queueName].Messages[:i], sqs.Queues[queueName].Messages[i+1:]...)

				sqs.Unlock()
				return &app.DeleteMessageResponse{"http://queue.amazonaws.com/doc/2012-11-05/", app.ResponseMetadata{RequestId: "00000000-0000-0000-0000-000000000001"}}
			}
		}
		log.Println("Receipt Handle not found")
	} else {
		log.Println("Queue not found")
	}
	sqs.Unlock()

	createErrorResponse(w, req, "MessageDoesNotExist")
}

func (sqs * SQS) DeleteQueue(args map[string]string) (*DeleteQueueResponse, error) {
	vars := mux.Vars(req)
	queueName := vars["queueName"]

	log.Println("Deleting Queue:", queueName)
	sqs.Lock()
	delete(sqs.Queues, queueName)
	sqs.Unlock()

	// Create, encode/xml and send response
	return &app.DeleteQueueResponse{"http://queue.amazonaws.com/doc/2012-11-05/", app.ResponseMetadata{RequestId: "00000000-0000-0000-0000-000000000000"}}
}

func (sqs * SQS) PurgeQueue(args map[string]string) (*PurgeQueueResponse, error) {
	vars := mux.Vars(req)
	queueName := vars["queueName"]

	log.Println("Purging Queue:", queueName)

	sqs.Lock()
	if _, ok := sqs.Queues[queueName]; ok {
		sqs.Queues[queueName].Messages = nil
		return &app.PurgeQueueResponse{"http://queue.amazonaws.com/doc/2012-11-05/", app.ResponseMetadata{RequestId: "00000000-0000-0000-0000-000000000000"}}
	} else {
		log.Println("Purge Queue:", queueName, ", queue does not exist!!!")
		createErrorResponse(w, req, "QueueNotFound")
	}
	sqs.Unlock()
}

func (sqs * SQS) GetQueueUrl(args map[string]string) (*GetQueueUrlResponse, error) {
	vars := mux.Vars(req)
	queueName := vars["queueName"]
	host := args["host"]
	if queue, ok := sqs.Queues[queueName]; ok {
		url := sqs.QueueUrl(queueName, host)
		log.Println("Get Queue URL:", queueName)
		return &GetQueueUrlResponse{
			Response: &NewResponse()
			Result: &app.GetQueueUrlResult{QueueUrl: url}
		}
	} else {
		log.Println("Get Queue URL:", queueName, ", queue does not exist!!!")
		createErrorResponse("QueueNotFound")
	}
}

func (sqs * SQS) GetQueueAttributes(args map[string]string) (*GetQueueAttributesResponse, error) {
	vars := mux.Vars(req)
	queueName := vars["queueName"]

	log.Println("Get Queue Attributes:", queueName)
	sqs.RLock()
	if queue, ok := sqs.Queues[queueName]; ok {
		// Create, encode/xml and send response
		attribs := make([]app.Attribute, 0, 0)
		attr := app.Attribute{Name: "VisibilityTimeout", Value: strconv.Itoa(queue.TimeoutSecs)}
		attribs = append(attribs, attr)
		attr = app.Attribute{Name: "DelaySeconds", Value: "0"}
		attribs = append(attribs, attr)
		attr = app.Attribute{Name: "ReceiveMessageWaitTimeSeconds", Value: strconv.Itoa(queue.ReceiveWaitTimeSecs)}
		attribs = append(attribs, attr)
		attr = app.Attribute{Name: "ApproximateNumberOfMessages", Value: strconv.Itoa(len(queue.Messages))}
		attribs = append(attribs, attr)
		sqs.RLock()
		attr = app.Attribute{Name: "ApproximateNumberOfMessagesNotVisible", Value: strconv.Itoa(numberOfHiddenMessagesInQueue(*queue))}
		sqs.RUnlock()
		attribs = append(attribs, attr)
		attr = app.Attribute{Name: "CreatedTimestamp", Value: "0000000000"}
		attribs = append(attribs, attr)
		attr = app.Attribute{Name: "LastModifiedTimestamp", Value: "0000000000"}
		attribs = append(attribs, attr)
		attr = app.Attribute{Name: "QueueArn", Value: queue.Arn}
		attribs = append(attribs, attr)

		deadLetterTargetArn := ""
		if queue.DeadLetterQueue != nil {
			deadLetterTargetArn = queue.DeadLetterQueue.Name
		}
		attr = app.Attribute{Name: "RedrivePolicy", Value: fmt.Sprintf(`{"maxReceiveCount": "%d", "deadLetterTargetArn":"%s"}`, queue.MaxReceiveCount, deadLetterTargetArn)}
		attribs = append(attribs, attr)

		return &GetQueueAttributesResponse{
			Response: &NewResponse(),
			Result: &GetQueueAttributesResult{
				Attrs: attribs,
			},
		}
	} else {
		log.Println("Get Queue URL:", queueName, ", queue does not exist!!!")
		createErrorResponse("QueueNotFound")
	}
	sqs.RUnlock()
}

func (sqs * SQS) SetQueueAttributes(args map[string]string) (*SetQueueAttributesResponse, error) {
	vars := mux.Vars(req)
	queueName := vars["queueName"]

	log.Println("Set Queue Attributes:", queueName)
	sqs.Lock()
	if queue, ok := sqs.Queues[queueName]; ok {
		if err := validateAndSetQueueAttributes(queue, req.Form); err != nil {
			sqs.Unlock()
			return createErrorResponse(err.Error())
		}
		return &SetQueueAttributesResponse{
			Response: &NewResponse(),
		}
	} else {
		log.Println("Get Queue URL:", queueName, ", queue does not exist!!!")
		return createErrorResponse("QueueNotFound")
	}
	sqs.Unlock()
}

func (sqs * SQS) QueueUrl(queueName string, host string) string {
	return fmt.Sprintf("https://%s.%s/%s/%s", sqs.Region, host, sqs.AccountID, queueName)
}

func (sqs * SQS) QueueArn(queueName string) string {
	return fmt.Sprintf("arn:aws:sqs:%s:%s:%s", sqs.Region, sqs.AccountID, queueName)
}

func getMessageResult(m *app.Message) *app.ResultMessage {
	msgMttrs := []*app.ResultMessageAttribute{}
	for _, attr := range m.MessageAttributes {
		msgMttrs = append(msgMttrs, getMessageAttributeResult(&attr))
	}

	attrsMap := map[string]string{
		"ApproximateFirstReceiveTimestamp": fmt.Sprintf("%d", m.ReceiptTime.UnixNano()/int64(time.Millisecond)),
		"SenderId":                         app.CurrentEnvironment.AccountID,
		"ApproximateReceiveCount":          fmt.Sprintf("%d", m.NumberOfReceives+1),
		"SentTimestamp":                    fmt.Sprintf("%d", time.Now().UTC().UnixNano()/int64(time.Millisecond)),
	}

	var attrs []*app.ResultAttribute
	for k, v := range attrsMap {
		attrs = append(attrs, &app.ResultAttribute{
			Name:  k,
			Value: v,
		})
	}

	return &app.ResultMessage{
		MessageId:              m.Uuid,
		Body:                   m.MessageBody,
		ReceiptHandle:          m.ReceiptHandle,
		MD5OfBody:              common.GetMD5Hash(string(m.MessageBody)),
		MD5OfMessageAttributes: m.MD5OfMessageAttributes,
		MessageAttributes:      msgMttrs,
		Attributes:             attrs,
	}
}
