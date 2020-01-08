package app

/*** List Topics Response */
type TopicArnResult struct {
	TopicArn string `xml:"TopicArn"`
}

type TopicNamestype struct {
	Member []TopicArnResult `xml:"member"`
}

type ListTopicsResult struct {
	Topics TopicNamestype `xml:"Topics"`
}

type ListTopicsResponse struct {
	Response
	Result   ListTopicsResult `xml:"ListTopicsResult"`
}

/*** Create Topic Response */
type CreateTopicResult struct {
	TopicArn string `xml:"TopicArn"`
}

type CreateTopicResponse struct {
	Response
	Result   CreateTopicResult `xml:"CreateTopicResult"`
}

/*** Create Subscription ***/
type SubscribeResult struct {
	SubscriptionArn string `xml:"SubscriptionArn"`
}

type SubscribeResponse struct {
	Response
	Result   SubscribeResult  `xml:"SubscribeResult"`
}

/*** ConfirmSubscriptionResponse ***/
type ConfirmSubscriptionResponse struct {
	Response
	Result   SubscribeResult  `xml:"ConfirmSubscriptionResult"`
}

/***  Set Subscription Response ***/

type SetSubscriptionAttributesResponse struct {
	Response
}

/*** Get Subscription Attributes ***/
type GetSubscriptionAttributesResult struct {
	SubscriptionAttributes SubscriptionAttributes `xml:"Attributes,omitempty"`
}

type SubscriptionAttributes struct {
	/* SubscriptionArn, FilterPolicy */
	Entries []SubscriptionAttributeEntry `xml:"entry,omitempty"`
}

type SubscriptionAttributeEntry struct {
	Key   string `xml:"key,omitempty"`
	Value string `xml:"value,omitempty"`
}

type GetSubscriptionAttributesResponse struct {
	Response
	Result   GetSubscriptionAttributesResult `xml:"GetSubscriptionAttributesResult"`
}

/*** List Subscriptions Response */
type TopicMemberResult struct {
	TopicArn        string `xml:"TopicArn"`
	Protocol        string `xml:"Protocol"`
	SubscriptionArn string `xml:"SubscriptionArn"`
	Owner           string `xml:"Owner"`
	Endpoint        string `xml:"Endpoint"`
}

type TopicSubscriptions struct {
	Member []TopicMemberResult `xml:"member"`
}

type ListSubscriptionsResult struct {
	Subscriptions TopicSubscriptions `xml:"Subscriptions"`
}

type ListSubscriptionsResponse struct {
	Response
	Result   ListSubscriptionsResult `xml:"ListSubscriptionsResult"`
}

/*** List Subscriptions By Topic Response */

type ListSubscriptionsByTopicResult struct {
	Subscriptions TopicSubscriptions `xml:"Subscriptions"`
}

type ListSubscriptionsByTopicResponse struct {
	Response
	Result   ListSubscriptionsResult `xml:"ListSubscriptionsResult"`
}

/*** Publish ***/

type PublishResult struct {
	MessageId string `xml:"MessageId"`
}

type PublishResponse struct {
	Response
	Result   PublishResult    `xml:"PublishResult"`
}

/*** Unsubscribe ***/
type UnsubscribeResponse struct {
	Response
}

/*** Delete Topic ***/
type DeleteTopicResponse struct {
	Response
}
