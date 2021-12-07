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
    - [Using configured XSLT](#using-configured-xslt)
    - [Using per event XSLT](#using-per-event-xslt)
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
- When the `allowPerEventXslt` flag is set it also accepts an `io.triggermesh.xslt.transform` event type that must include an `xml` and `xslt` element for the transformation.

```json
{
  "xml": "<?xml version=\"1.0\" encoding=\"UTF-8\"?><element1>..."
  "xslt": "<?xml version=\"1.0\" encoding=\"UTF-8\"?><xsl:stylesheet version=\"1.0\">..."
}
```

## Example

You can find an example at the [samples folder](../../config/samples/flows/xslttransform) which contains:

- a broker.
- an XSLT transformation.
- an event-display service.
- a filtered trigger that sends CloudEvents typed `xml.document` to the XSLT transformation.
- a filtered trigger that sends CloudEvents typed `io.triggermesh.xslt.transform` to the XSLT transformation.
- an unfiltered trigger that sends all CloudEvents to the event-display service.
- a curl pod for interacting.

The example is setup with an XSLT transformation but also allows overriding the XSLT at each event if the type `io.triggermesh.xslt.transform` is used.

```console
kubectl apply -f config/samples/flows/xslttransform
```

### Using configured XSLT

Use the curl command to send a CloudEvent to be transformed to the broker.

```console
kubectl exec -ti curl -- curl -v "http://broker-ingress.knative-eventing.svc.cluster.local/default/demo" \
  -H "Ce-Specversion: 1.0" \
  -H "Ce-Type: xml.document" \
  -H "Ce-Source: curl.shell" \
  -H "Content-Type: application/xml" \
  -H "Ce-Id: 1234-abcd" \
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

Check output at the event-display pod.

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

### Using per event XSLT

Send an `io.triggermesh.xslt.transform` CloudEvent that includesboth XML and XSLT contents.

```console
kubectl exec -ti curl -- curl -v 'http://broker-ingress.knative-eventing.svc.cluster.local/default/demo' \
  -H 'Ce-Specversion: 1.0' \
  -H 'Ce-Type: io.triggermesh.xslt.transform' \
  -H 'Ce-Source: curl.shell' \
  -H 'Content-Type: application/json' \
  -H 'Ce-Id: 1234-abcd' \
  -d '{
  "xml": "<guitars><guitar><brand>Framus</brand><model>AK 1974</model><year>2012</year></guitar><guitar><brand>Fender</brand><model>Jaguar</model><year>2010</year></guitar><guitar><brand>Haar</brand><model>Telecaster</model><year>2019</year></guitar><guitar><brand>Hohner</brand><model>L75</model><year>1992</year></guitar></guitars>",
  "xslt": "<?xml version=\"1.0\" encoding=\"UTF-8\"?><xsl:stylesheet version=\"1.0\" xmlns:xsl=\"http://www.w3.org/1999/XSL/Transform\"><xsl:template match=\"/\"><output><xsl:for-each select=\"guitars/guitar\"><xsl:sort select=\"year\"/><item><xsl:value-of select=\"year\"/><xsl:text> </xsl:text><xsl:value-of select=\"brand\"/><xsl:text> </xsl:text><xsl:value-of select=\"model\"/></item></xsl:for-each></output></xsl:template></xsl:stylesheet>"
}
'
```

Check event-display output for the transformed output.

```console
kubectl logs -l serving.knative.dev/service=event-display -c user-container

☁️  cloudevents.Event
Context Attributes,
  specversion: 1.0
  type: io.triggermesh.xslt.transform.response
  source: xslttransform-adapter
  id: 5d3b88c9-dd8b-46a6-9259-a63b56281647
  time: 2021-12-07T09:31:45.83337697Z
  datacontenttype: application/xml
Extensions,
  category: success
  knativearrivaltime: 2021-12-07T09:31:45.84042863Z
Data,
  <?xml version="1.0"?>
<output>
  <item>1992 Hohner L75</item>
  <item>2010 Fender Jaguar</item>
  <item>2012 Framus AK 1974</item>
  <item>2019 Haar Telecaster</item>
</output>
```

## Developing

When building XsltTransform container image, make sure that [library depencencies are satisfied](../../hack/images/xslttransform).
