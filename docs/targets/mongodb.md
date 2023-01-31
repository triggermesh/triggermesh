# MongoDBTarget

The MongoDBTarget exposes several methods, via event types, that can be used to interact with the MongoDB database.

# Interacting with the Event Target
## Arbitrary Event Types
The mongoDBTarget supports accepting arbitrary event types. These events will be inserted into the MongoDB database/collection specified in the `defaultDatabase` and `defaultCollection` spec fields.

So if one were to send the following event:

```cmd
curl -v  http://localhost:8080 \
       -X POST \
       -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
       -H "Ce-Specversion: 1.0" \
       -H "Ce-Type: io.triggermesh.sample.event" \
       -H "Ce-Source: sample" \
       -H "Content-Type: application/json" \
       -d '{"example":"event"}'
```

We would expect a similar entry in the MongoDB database collection:

```
{"_id":{"$oid":"62029b4e20372fe01225194d"},"example":"event"}
```

## Pre-defined Event Types

Use these event types and associated payloads to interact with the MongoDB Target.

### io.triggermesh.mongodb.insert

Events of this type intend to post a single key:value pair to MongoDB

#### Example CE posting an event of type "io.triggermesh.mongodb.insert"

```cmd
curl -v  http://localhost:8080 \
       -X POST \
       -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
       -H "Ce-Specversion: 1.0" \
       -H "Ce-Type: io.triggermesh.mongodb.insert" \
       -H "Ce-Source: sample/source" \
       -H "Content-Type: application/json" \
       -d '{"database":"test","collection": "test","jsonMessage":{"test":"testdd1","test2":"test3"}}'
```

#### This type expects a JSON payload with the following properties:

| Name  |  Type |  Comment |
|---|---|---|
| **database** | string | The name of the database.  |
| **collection** | string | The value of the collection. |
| **key** | string | This value will be used to assing a Key.  |
| **jsonMessage** | map[string]string | This value will be used to assing a Value. |

**Note** the `database` and `collection` fields are not required. If not provided, the `defaultDatabase` and `defaultCollection` spec fields will be used.

### io.triggermesh.mongodb.update

Events of this type intend to update any pre-existing key:value pair(s). 

#### Example CE posting an event of type "io.triggermesh.mongodb.update"

```cmd
curl -v http://localhost:8080 \
       -X POST \
       -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
       -H "Ce-Specversion: 1.0" \
       -H "Ce-Type: io.triggermesh.mongodb.update" \
       -H "Ce-Source: sample/source" \
       -H "Content-Type: application/json" \
       -d '{"database":"test","collection": "test","searchKey":"test","searchValue":"testdd1","updateKey":"partstore","updateValue":"UP FOR GRABS"}'
```

#### This type expects a JSON payload with the following properties:

| Name  |  Type |  Comment |
|---|---|---|
| **database** | string | The name of the database.  |
| **collection** | string | The value of the collection. |
| **itemName** | string | This value will be used to assing a Key.  |
| **searchKey** | string | . |
| **searchValue** | string | .  |
| **updateKey** | string | .  |
| **updateValue** | string |. |

**Note** the `database` and `collection` fields are not required. If not provided, the `defaultDatabase` and `defaultCollection` spec fields will be used.

### io.triggermesh.mongodb.query.kv

Events of this type intend to query a MongoDB for any documents that contain a matching key/value pair. 

#### Example CE posting an event of type "io.triggermesh.mongodb.query.kv"

```cmd
curl -v http://localhost:8080 \
       -X POST \
       -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
       -H "Ce-Specversion: 1.0" \
       -H "Ce-Type: io.triggermesh.mongodb.query.kv" \
       -H "Ce-Source: sample/source" \
       -H "Content-Type: application/json" \
       -d '{"database":"test","Collection": "test","key":"partstore","value":"UP FOR GRABS"}'
```

#### This type expects a JSON payload with the following properties:

| Name  |  Type |  Comment |
|---|---|---|
| **database** | string | The name of the database.  |
| **collection** | string | The value of the collection. |
| **key** | string | the "Key" value to search  |
| **value** | string | the "Value" value to search |

**Note** the `database` and `collection` fields are not required. If not provided, the `defaultDatabase` and `defaultCollection` spec fields will be used.

##### Example response of  type "io.triggermesh.mongodb.query.kv"

```
Ce-Id: 60cee2e6-a3d0-4ff5-8157-b80ff3e8797b
Ce-Source: io.triggermesh.mongodb
Ce-Specversion: 1.0
Ce-Subject: query-result
Ce-Time: 2023-01-19T18:22:46.928999Z
Ce-Type: io.triggermesh.mongodb.query.kv.result
Content-Length: 96
Content-Type: application/json
Date: Thu, 19 Jan 2023 18:22:46 GMT

Connection #0 to host localhost left intact
[{"_id":"63c829397c2fdbfebdd93883","partstore":"UP FOR GRABS","test":"testdd1","test2":"test3"}]%   
```

# Local Development

To build and run this Target locally, run the following command(s):

```cmd
export MONGODB_SERVER_URL=mongodb+srv://<user>:<password>@<database_url>/myFirstDatabase
export MONGODB_DEFAULT_DATABASE=testdb
export MONGODB_DEFAULT_COLLECTION=testcol

go run cmd/mongodbtarget-adapter/main.go
```
