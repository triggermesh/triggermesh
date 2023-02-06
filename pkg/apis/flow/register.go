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

package flow

import "k8s.io/apimachinery/pkg/runtime/schema"

const (
	// GroupName is the name of the API group this package's resources belong to.
	GroupName = "flow.triggermesh.io"
)

var (
	// JQTransformationResource respresents a JQ transformation.
	JQTransformationResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "jqtransformations",
	}

	// SynchronizerResource respresents a Synchronizer.
	SynchronizerResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "synchronizers",
	}

	// TransformationResource respresents a Bumblebee transformation.
	TransformationResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "transformations",
	}

	// XMLToJSONTransformationResource respresents a XML to JSON transformation.
	XMLToJSONTransformationResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "xmltojsontansformations",
	}

	// XSLTTransformationResource respresents a XSLT transformation.
	XSLTTransformationResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "xslttransformations",
	}
)
