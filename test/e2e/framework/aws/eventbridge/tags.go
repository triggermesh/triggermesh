/*
Copyright 2022 TriggerMesh Inc.

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

package eventbridge

import (
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/eventbridge"
)

// tagsAsEventBridgeTags converts a map containing resource tags to a list of
// EventBridge resource tags.
func tagsAsEventBridgeTags(tags map[string]*string) []*eventbridge.Tag {
	ebTags := make([]*eventbridge.Tag, 0, len(tags))

	for k, v := range tags {
		ebTags = append(ebTags, &eventbridge.Tag{
			Key:   aws.String(k),
			Value: v,
		})
	}

	sort.Sort(tagSlice(ebTags))

	return ebTags
}

type tagSlice []*eventbridge.Tag

func (t tagSlice) Len() int           { return len(t) }
func (t tagSlice) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t tagSlice) Less(i, j int) bool { return *t[i].Key < *t[j].Key }
