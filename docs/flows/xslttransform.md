# XSLT Transformations

The XSLT transformation component transforms a XML document using XSLT.

## Contents

- [XSLT Transformations](#xslt-transformations)
  - [Contents](#contents)
  - [Prerequisites](#prerequisites)
  - [Usage](#usage)
    - [Setup](#setup)
    - [CloudEvents](#cloudevents)
  - [Example](#example)
  - [Developing](#developing)

## Prerequisites

The component needs to be configured with a valid XSLT document.
For the examples at this document we will use this XSLT:

being transformed by this XSLT document:

```xml
<xsl:stylesheet version="1.0"	xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="tests">
    <output>
      <xsl:apply-templates select="test">
        <xsl:sort select="data/el1"/>
        <xsl:sort select="data/el2"/>
      </xsl:apply-templates>
    </output>
  </xsl:template>

  <xsl:template match="test">
    <item>
      <xsl:value-of select="data/el1"/>
      <xsl:value-of select="data/el2"/>
    </item>
  </xsl:template>
</xsl:stylesheet>
```

in order to transform this input:

```xml
<tests>
  <test>
    <data>
      <el1>A</el1>
      <el2>1</el2>
    </data>
  </test>
  <test>
    <data>
			<el1>B</el1>
			<el2>2</el2>
    </data>
  </test>
  <test>
    <data>
			<el1>C</el1>
			<el2>3</el2>
    </data>
  </test>
</tests>
```

and expect this output:

```xml
<?xml version="1.0"?>
<output>
  <item>A1</item>
  <item>B2</item>
  <item>C3</item>
</output>
```
## Usage

### Setup

The API can be inspected using `kubectl explain` for each field level:

```console
kubectl explain  xslttransform.spec
KIND:     XsltTransform
VERSION:  flow.triggermesh.io/v1alpha1

RESOURCE: spec <Object>

DESCRIPTION:
     Desired state of the TriggerMesh component.

FIELDS:
   allowPerEventXslt    <boolean>
     Whether the XSLT informed at the spec can be overriden at each CloudEvent.

   xslt <Object>
     XSLT used to transform incoming CloudEvents.
```

### CloudEvents

- The transformation accepts any CloudEvent whose media type is `application/xml`.
- On success the output event will contain the transformed XML keeping the `application/xml` media type, and appending a `.response` suffix to the incoming CloudEvent type.

## Example

You can find an example at the [samples folder](../../config/samples/flows/xslttransform) which contains:

- a broker.
- an XSLT transformation.
- an event-display service.
- a filtered trigger that sends CloudEvents typed `xml.document` to the XSLT transformation.
- a filtered trigger that sends CloudEvents typed `xml.document.response` to the event-display service.
- a curl pod for interacting.

```console
kubectl apply -f config/samples/flows/xslttransform
```

Use the curl command to send a CloudEvent to be transformed to the broker.

```console
kubectl exec -ti curl -- curl -v "http://broker-ingress.knative-eventing.svc.cluster.local/default/demo" \
  -H "Ce-Specversion: 1.0" \
  -H "Ce-Type: xml.document" \
  -H "Ce-Source: curl.shell" \
  -H "Content-Type: application/xml" \
  -H "Ce-Id: conformance-test-nack" \
  -d "<tests>
  <test>
    <data>
      <el1>A</el1>
      <el2>1</el2>
    </data>
  </test>
  <test>
    <data>
			<el1>B</el1>
			<el2>2</el2>
    </data>
  </test>
  <test>
    <data>
			<el1>C</el1>
			<el2>3</el2>
    </data>
  </test>
</tests>"
```

Check output at the event-display pod

```console
kubectl logs -l serving.knative.dev/service=event-display -c user-container

☁️  cloudevents.Event
Context Attributes,
  specversion: 1.0
  type: xml.document.response
  source: xslttransform-adapter
  id: 2a40265b-0bca-43d3-aa97-7c72fdff0813
  time: 2021-12-01T10:26:02.087630067Z
  datacontenttype: application/json
Extensions,
  category: success
  knativearrivaltime: 2021-12-01T10:26:02.093007305Z
Data,
  <?xml version="1.0"?>
<output>
  <item>A1</item>
  <item>B2</item>
  <item>C3</item>
</output>
```

## Developing

When building XsltTransform container image, make sure that [library depencencies are satisfied](../../hack/images/xslttransform).
