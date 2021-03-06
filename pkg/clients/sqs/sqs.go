/*
Copyright 2019 The Crossplane Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sqs

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/go-cmp/cmp"

	"github.com/crossplane/provider-aws/apis/applicationintegration/v1alpha1"
	awsclients "github.com/crossplane/provider-aws/pkg/clients"
)

const (
	// QueueNotFound is the code that is returned by AWS when the given QueueURL is not valid
	QueueNotFound = "AWS.SimpleQueueService.NonExistentQueue"
)

// Client defines Queue client operations
type Client interface {
	CreateQueueRequest(input *sqs.CreateQueueInput) sqs.CreateQueueRequest
	DeleteQueueRequest(input *sqs.DeleteQueueInput) sqs.DeleteQueueRequest
	TagQueueRequest(input *sqs.TagQueueInput) sqs.TagQueueRequest
	ListQueueTagsRequest(*sqs.ListQueueTagsInput) sqs.ListQueueTagsRequest
	GetQueueAttributesRequest(*sqs.GetQueueAttributesInput) sqs.GetQueueAttributesRequest
	SetQueueAttributesRequest(input *sqs.SetQueueAttributesInput) sqs.SetQueueAttributesRequest
	UntagQueueRequest(input *sqs.UntagQueueInput) sqs.UntagQueueRequest
	GetQueueUrlRequest(input *sqs.GetQueueUrlInput) sqs.GetQueueUrlRequest
}

// NewClient returns a new SQS Client.
func NewClient(cfg aws.Config) Client {
	return sqs.New(cfg)
}

// GenerateCreateAttributes returns a map of queue attributes for Create operation
func GenerateCreateAttributes(p *v1alpha1.QueueParameters) map[string]string {
	m := GenerateQueueAttributes(p)
	if p.FIFOQueue != nil {
		m[v1alpha1.AttributeFifoQueue] = fmt.Sprint(*p.FIFOQueue)
	}
	return m
}

// GenerateUpdateAttributes returns a map of queue attributes for update operation
func GenerateUpdateAttributes(p *v1alpha1.QueueParameters) map[string]string {
	return GenerateQueueAttributes(p)
}

// GenerateQueueAttributes returns a map of queue attributes
func GenerateQueueAttributes(p *v1alpha1.QueueParameters) map[string]string { // nolint:gocyclo
	m := map[string]string{}
	if p.DelaySeconds != nil {
		m[v1alpha1.AttributeDelaySeconds] = strconv.FormatInt(aws.Int64Value(p.DelaySeconds), 10)
	}
	if p.MaximumMessageSize != nil {
		m[v1alpha1.AttributeMaximumMessageSize] = strconv.FormatInt(aws.Int64Value(p.MaximumMessageSize), 10)
	}
	if p.MessageRetentionPeriod != nil {
		m[v1alpha1.AttributeMessageRetentionPeriod] = strconv.FormatInt(aws.Int64Value(p.MessageRetentionPeriod), 10)
	}
	if p.ReceiveMessageWaitTimeSeconds != nil {
		m[v1alpha1.AttributeReceiveMessageWaitTimeSeconds] = strconv.FormatInt(aws.Int64Value(p.ReceiveMessageWaitTimeSeconds), 10)
	}
	if p.VisibilityTimeout != nil {
		m[v1alpha1.AttributeVisibilityTimeout] = strconv.FormatInt(aws.Int64Value(p.VisibilityTimeout), 10)
	}
	if p.KMSMasterKeyID != nil {
		m[v1alpha1.AttributeKmsMasterKeyID] = aws.StringValue(p.KMSMasterKeyID)
	}
	if p.KMSDataKeyReusePeriodSeconds != nil {
		m[v1alpha1.AttributeKmsDataKeyReusePeriodSeconds] = strconv.FormatInt(aws.Int64Value(p.KMSDataKeyReusePeriodSeconds), 10)
	}
	if p.ReceiveMessageWaitTimeSeconds != nil {
		m[v1alpha1.AttributeReceiveMessageWaitTimeSeconds] = strconv.FormatInt(aws.Int64Value(p.ReceiveMessageWaitTimeSeconds), 10)
	}

	if p.RedrivePolicy != nil && aws.StringValue(p.RedrivePolicy.DeadLetterQueueARN) != "" {
		val, err := json.Marshal(p.RedrivePolicy)
		if err == nil {
			m[v1alpha1.AttributeRedrivePolicy] = string(val)
		}
	}
	return m
}

// GenerateQueueTags returns a map of queue tags
func GenerateQueueTags(tags []v1alpha1.Tag) map[string]string {
	if len(tags) != 0 {
		m := map[string]string{}
		for _, val := range tags {
			m[val.Key] = aws.StringValue(val.Value)
		}
		return m
	}
	return nil
}

// IsNotFound checks if the error returned by AWS API says that the queue being probed doesn't exist
func IsNotFound(err error) bool {
	if awsErr, ok := err.(awserr.Error); ok {
		if awsErr.Code() == QueueNotFound {
			return true
		}
	}

	return false
}

// LateInitialize fills the empty fields in *v1alpha1.QueueParameters with
// the values seen in queue.Attributes
func LateInitialize(in *v1alpha1.QueueParameters, attributes map[string]string, tags map[string]string) {
	if in.Tags == nil && tags != nil {
		for k, v := range tags {
			in.Tags = append(in.Tags, v1alpha1.Tag{Key: k, Value: aws.String(v)})
		}
	}

	if len(attributes) == 0 {
		return
	}
	in.DelaySeconds = awsclients.LateInitializeInt64Ptr(in.DelaySeconds, int64Ptr(attributes[v1alpha1.AttributeDelaySeconds]))
	in.KMSDataKeyReusePeriodSeconds = awsclients.LateInitializeInt64Ptr(in.KMSDataKeyReusePeriodSeconds, int64Ptr(attributes[v1alpha1.AttributeKmsDataKeyReusePeriodSeconds]))
	in.MaximumMessageSize = awsclients.LateInitializeInt64Ptr(in.MaximumMessageSize, int64Ptr(attributes[v1alpha1.AttributeMaximumMessageSize]))
	in.MessageRetentionPeriod = awsclients.LateInitializeInt64Ptr(in.MessageRetentionPeriod, int64Ptr(attributes[v1alpha1.AttributeMessageRetentionPeriod]))
	in.ReceiveMessageWaitTimeSeconds = awsclients.LateInitializeInt64Ptr(in.ReceiveMessageWaitTimeSeconds, int64Ptr(attributes[v1alpha1.AttributeReceiveMessageWaitTimeSeconds]))
	in.VisibilityTimeout = awsclients.LateInitializeInt64Ptr(in.VisibilityTimeout, int64Ptr(attributes[v1alpha1.AttributeVisibilityTimeout]))
	if in.KMSMasterKeyID == nil && attributes[v1alpha1.AttributeKmsMasterKeyID] != "" {
		in.KMSMasterKeyID = aws.String(attributes[v1alpha1.AttributeKmsMasterKeyID])
	}

	if attributes[v1alpha1.AttributeDeadLetterQueueARN] != "" || attributes[v1alpha1.AttributeMaxReceiveCount] != "" {
		in.RedrivePolicy = &v1alpha1.RedrivePolicy{}
		in.RedrivePolicy.MaxReceiveCount = awsclients.LateInitializeInt64Ptr(in.RedrivePolicy.MaxReceiveCount, int64Ptr(attributes[v1alpha1.AttributeMaxReceiveCount]))
		in.RedrivePolicy.DeadLetterQueueARN = awsclients.LateInitializeStringPtr(in.RedrivePolicy.DeadLetterQueueARN, aws.String(attributes[v1alpha1.AttributeDeadLetterQueueARN]))
	}
}

// IsUpToDate checks whether there is a change in any of the modifiable fields.
func IsUpToDate(p v1alpha1.QueueParameters, attributes map[string]string, tags map[string]string) bool { // nolint:gocyclo
	if len(p.Tags) != len(tags) {
		return false
	}

	for _, tag := range p.Tags {
		pVal, ok := tags[tag.Key]
		if !ok || !strings.EqualFold(pVal, aws.StringValue(tag.Value)) {
			return false
		}
	}

	if aws.Int64Value(p.DelaySeconds) != int64Value(attributes[v1alpha1.AttributeDelaySeconds]) {
		return false
	}
	if aws.Int64Value(p.KMSDataKeyReusePeriodSeconds) != int64Value(attributes[v1alpha1.AttributeKmsDataKeyReusePeriodSeconds]) {
		return false
	}
	if aws.Int64Value(p.MaximumMessageSize) != int64Value(attributes[v1alpha1.AttributeMaximumMessageSize]) {
		return false
	}
	if aws.Int64Value(p.MessageRetentionPeriod) != int64Value(attributes[v1alpha1.AttributeMessageRetentionPeriod]) {
		return false
	}
	if aws.Int64Value(p.ReceiveMessageWaitTimeSeconds) != int64Value(attributes[v1alpha1.AttributeReceiveMessageWaitTimeSeconds]) {
		return false
	}
	if aws.Int64Value(p.VisibilityTimeout) != int64Value(attributes[v1alpha1.AttributeVisibilityTimeout]) {
		return false
	}
	if !cmp.Equal(aws.StringValue(p.KMSMasterKeyID), attributes[v1alpha1.AttributeKmsMasterKeyID]) {
		return false
	}

	if p.RedrivePolicy != nil {
		if !cmp.Equal(aws.StringValue(p.RedrivePolicy.DeadLetterQueueARN), attributes[v1alpha1.AttributeDeadLetterQueueARN]) {
			return false
		}
		if aws.Int64Value(p.RedrivePolicy.MaxReceiveCount) != int64Value(attributes[v1alpha1.AttributeMaxReceiveCount]) {
			return false
		}
	}

	return true
}

// TagsDiff returns the tags added and removed from spec when compared to the AWS SQS tags.
func TagsDiff(sqsTags map[string]string, specTags []v1alpha1.Tag) (removed, added map[string]string) {
	newTags := GenerateQueueTags(specTags)

	removed = map[string]string{}
	for k, v := range sqsTags {
		if _, ok := newTags[k]; !ok {
			removed[k] = v
		}
	}

	added = map[string]string{}
	for k, newV := range newTags {
		if oldV, ok := sqsTags[k]; !ok || oldV != newV {
			added[k] = newV
		}
	}
	return
}

func int64Value(s string) int64 {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return v
}

func int64Ptr(s string) *int64 {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return nil
	}
	return &v
}
