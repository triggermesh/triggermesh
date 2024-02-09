
I inserted: 
```
{"_id":{"$oid":"646794d7bfe0edad21ef6346"},"hello":"world"}
```

and recieved at my sink:

```
Context Attributes,
  specversion: 1.0
  type: io.triggermesh.mongodb.event
  source: default/sample-mongodbsource
  id: edfb138c-de86-48f0-a0b3-a7648dd773ff
  time: 2023-05-19T15:26:07.430860192Z
  datacontenttype: application/json
Data,
  {
    "_id": {
      "_data": "826467950F000000092B022C0100296E5A100402564FD535C0466984FB5C4D928AA3E146645F69640064646794D7BFE0EDAD21EF63460004"
    },
    "clusterTime": {
      "T": 1684509967,
      "I": 9
    },
    "documentKey": {
      "_id": "646794d7bfe0edad21ef6346"
    },
    "fullDocument": {
      "_id": "646794d7bfe0edad21ef6346",
      "hello": "world"
    },
    "ns": {
      "coll": "testcol",
      "db": "testdb"
    },
    "operationType": "insert",
    "wallTime": "2023-05-19T15:26:07.385Z"
  }
```

I then deleted this same entry and recieved this at my sink:

```
Context Attributes,
  specversion: 1.0
  type: io.triggermesh.mongodb.event
  source: default/sample-mongodbsource
  id: 7ac9d111-6382-40e6-b7e7-c1f2a1cb72d1
  time: 2023-05-19T15:26:54.502577722Z
  datacontenttype: application/json
Data,
  {
    "_id": {
      "_data": "826467953E000000142B022C0100296E5A100402564FD535C0466984FB5C4D928AA3E146645F69640064646794D7BFE0EDAD21EF63460004"
    },
    "clusterTime": {
      "T": 1684510014,
      "I": 20
    },
    "documentKey": {
      "_id": "646794d7bfe0edad21ef6346"
    },
    "ns": {
      "coll": "testcol",
      "db": "testdb"
    },
    "operationType": "delete",
    "wallTime": "2023-05-19T15:26:54.465Z"
  }
```

Updating an entry:

```
Context Attributes,
  specversion: 1.0
  type: io.triggermesh.mongodb.event
  source: default/sample-mongodbsource
  id: 3f6d1a86-f989-4983-be9b-63e4c2f640bf
  time: 2023-05-19T15:27:41.193043741Z
  datacontenttype: application/json
Data,
  {
    "_id": {
      "_data": "826467956D000000042B022C0100296E5A100402564FD535C0466984FB5C4D928AA3E146645F69640064644A89AC1386450E1516BF0A0004"
    },
    "clusterTime": {
      "T": 1684510061,
      "I": 4
    },
    "documentKey": {
      "_id": "644a89ac1386450e1516bf0a"
    },
    "ns": {
      "coll": "testcol",
      "db": "testdb"
    },
    "operationType": "update",
    "updateDescription": {
      "removedFields": [],
      "truncatedArrays": [],
      "updatedFields": {
        "hello": "space"
      }
    },
    "wallTime": "2023-05-19T15:27:41.154Z"
  }
```