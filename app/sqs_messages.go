package app

/*** List Queues */
type ListQueuesResult struct {
	QueueUrl []string `xml:"QueueUrl"`
}

type ListQueuesResponse struct {
	Response
	Result   ListQueuesResult `xml:"ListQueuesResult"`
}

/*** Create Queue Response */
type CreateQueueResult struct {
	QueueUrl string `xml:"QueueUrl"`
}

type CreateQueueResponse struct {
	Response
	Result   CreateQueueResult `xml:"CreateQueueResult"`
}

/*** Send Message */

type SendMessageResult struct {
	MD5OfMessageAttributes string `xml:"MD5OfMessageAttributes"`
	MD5OfMessageBody       string `xml:"MD5OfMessageBody"`
	MessageId              string `xml:"MessageId"`
	SequenceNumber         string `xml:"SequenceNumber"`
}

type SendMessageResponse struct {
	Response
	Result   SendMessageResult `xml:"SendMessageResult"`
}

/*** Receive Message */

type ResultMessage struct {
	MessageId              string                    `xml:"MessageId,omitempty"`
	ReceiptHandle          string                    `xml:"ReceiptHandle,omitempty"`
	MD5OfBody              string                    `xml:"MD5OfBody,omitempty"`
	Body                   []byte                    `xml:"Body,omitempty"`
	MD5OfMessageAttributes string                    `xml:"MD5OfMessageAttributes,omitempty"`
	MessageAttributes      []*ResultMessageAttribute `xml:"MessageAttribute,omitempty"`
	Attributes             []*ResultAttribute        `xml:"Attribute,omitempty"`
}

type ResultMessageAttributeValue struct {
	DataType    string `xml:"DataType,omitempty"`
	StringValue string `xml:"StringValue,omitempty"`
	BinaryValue string `xml:"BinaryValue,omitempty"`
}

type ResultMessageAttribute struct {
	Name  string                       `xml:"Name,omitempty"`
	Value *ResultMessageAttributeValue `xml:"Value,omitempty"`
}

type ResultAttribute struct {
	Name  string `xml:"Name,omitempty"`
	Value string `xml:"Value,omitempty"`
}

type ReceiveMessageResult struct {
	Message []*ResultMessage `xml:"Message,omitempty"`
}

type ReceiveMessageResponse struct {
	Response
	Result   ReceiveMessageResult `xml:"ReceiveMessageResult"`
}

type ChangeMessageVisibilityResponse struct {
	Response
}

/*** Delete Message */
type DeleteMessageResponse struct {
	Response
}

/*** Delete Queue */
type DeleteQueueResponse struct {
	Response
}

/*** Delete Message Batch */
type DeleteMessageBatchResultEntry struct {
	Id string `xml:"Id"`
}

type DeleteMessageBatchResult struct {
	Entry []DeleteMessageBatchResultEntry `xml:"DeleteMessageBatchResultEntry"`
	Error []BatchResultErrorEntry         `xml:"BatchResultErrorEntry,omitempty"`
}

type DeleteMessageBatchResponse struct {
	Response
	Result   DeleteMessageBatchResult `xml:"DeleteMessageBatchResult"`
}

/*** Send Message Batch */
type SendMessageBatchResultEntry struct {
	Id                     string `xml:"Id"`
	MessageId              string `xml:"MessageId"`
	MD5OfMessageBody       string `xml:"MD5OfMessageBody,omitempty"`
	MD5OfMessageAttributes string `xml:"MD5OfMessageAttributes,omitempty"`
	SequenceNumber         string `xml:"SequenceNumber"`
}

type BatchResultErrorEntry struct {
	Code        string `xml:"Code"`
	Id          string `xml:"Id"`
	Message     string `xml:"Message,omitempty"`
	SenderFault bool   `xml:"SenderFault"`
}

type SendMessageBatchResult struct {
	Entry []SendMessageBatchResultEntry `xml:"SendMessageBatchResultEntry"`
	Error []BatchResultErrorEntry       `xml:"BatchResultErrorEntry,omitempty"`
}

type SendMessageBatchResponse struct {
	Response
	Result   SendMessageBatchResult `xml:"SendMessageBatchResult"`
}

/*** Purge Queue */
type PurgeQueueResponse struct {
	Response
}

/*** Get Queue Url */
type GetQueueUrlResult struct {
	QueueUrl string `xml:"QueueUrl,omitempty"`
}

type GetQueueUrlResponse struct {
	Response
	Result   GetQueueUrlResult `xml:"GetQueueUrlResult"`
}

/*** Get Queue Attributes ***/
type Attribute struct {
	Name  string `xml:"Name,omitempty"`
	Value string `xml:"Value,omitempty"`
}

type GetQueueAttributesResult struct {
	/* VisibilityTimeout, DelaySeconds, ReceiveMessageWaitTimeSeconds, ApproximateNumberOfMessages
	   ApproximateNumberOfMessagesNotVisible, CreatedTimestamp, LastModifiedTimestamp, QueueArn */
	Attrs []Attribute `xml:"Attribute,omitempty"`
}

type GetQueueAttributesResponse struct {
	Response
	Result   GetQueueAttributesResult `xml:"GetQueueAttributesResult"`
}

/*** Set Queue Attributes ***/
type SetQueueAttributesResponse struct {
	Response
}
