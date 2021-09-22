/*
Copyright (c) 2021 TriggerMesh Inc.

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

package jiratarget

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andygrunwald/go-jira"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	cetest "github.com/cloudevents/sdk-go/v2/client/test"
	logtesting "knative.dev/pkg/logging/testing"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
)

const (
	tID          = "abc123"
	tContentType = "application/json"
	tSource      = "test.source"

	tResSource = "source.response"

	tIssueID = "10002"
	tIssue   = `
	{
		"expand":"renderedFields,names,schema,transitions,operations,editmeta,changelog,versionedRepresentations",
		"id":"10002",
		"self":"http://www.example.com/jira/rest/api/2/issue/10002",
		"key":"EX-1",
		"fields":{
			 "watcher":{
					"self":"http://www.example.com/jira/rest/api/2/issue/EX-1/watchers",
					"isWatching":false,
					"watchCount":1,
					"watchers":[
						 {
								"self":"http://www.example.com/jira/rest/api/2/user?username=fred",
								"name":"fred",
								"displayName":"Fred F. User",
								"active":false
						 }
					]
			 },
			 "sub-tasks":[
					{
						 "id":"10000",
						 "type":{
								"id":"10000",
								"name":"",
								"inward":"Parent",
								"outward":"Sub-task"
						 },
						 "outwardIssue":{
								"id":"10003",
								"key":"EX-2",
								"self":"http://www.example.com/jira/rest/api/2/issue/EX-2",
								"fields":{
									 "status":{
											"iconUrl":"http://www.example.com/jira//images/icons/statuses/open.png",
											"name":"Open"
									 }
								}
						 }
					}
			 ],
			 "description":"example bug report",
			 "project":{
					"self":"http://www.example.com/jira/rest/api/2/project/EX",
					"id":"10000",
					"key":"EX",
					"name":"Example",
					"avatarUrls":{
						 "48x48":"http://www.example.com/jira/secure/projectavatar?size=large&pid=10000",
						 "24x24":"http://www.example.com/jira/secure/projectavatar?size=small&pid=10000",
						 "16x16":"http://www.example.com/jira/secure/projectavatar?size=xsmall&pid=10000",
						 "32x32":"http://www.example.com/jira/secure/projectavatar?size=medium&pid=10000"
					},
					"projectCategory":{
						 "self":"http://www.example.com/jira/rest/api/2/projectCategory/10000",
						 "id":"10000",
						 "name":"FIRST",
						 "description":"First Project Category"
					}
			 },
			 "comment":{
					"comments":[
						 {
								"self":"http://www.example.com/jira/rest/api/2/issue/10010/comment/10000",
								"id":"10000",
								"author":{
									 "self":"http://www.example.com/jira/rest/api/2/user?username=fred",
									 "name":"fred",
									 "displayName":"Fred F. User",
									 "active":false
								},
								"body":"Lorem ipsum dolor sit amet, consectetur adipiscing elit. Pellentesque eget venenatis elit. Duis eu justo eget augue iaculis fermentum. Sed semper quam laoreet nisi egestas at posuere augue semper.",
								"updateAuthor":{
									 "self":"http://www.example.com/jira/rest/api/2/user?username=fred",
									 "name":"fred",
									 "displayName":"Fred F. User",
									 "active":false
								},
								"created":"2016-03-16T04:22:37.356+0000",
								"updated":"2016-03-16T04:22:37.356+0000",
								"visibility":{
									 "type":"role",
									 "value":"Administrators"
								}
						 }
					]
			 },
			 "worklog":{
					"worklogs":[
						 {
								"self":"http://www.example.com/jira/rest/api/2/issue/10010/worklog/10000",
								"author":{
									 "self":"http://www.example.com/jira/rest/api/2/user?username=fred",
									 "name":"fred",
									 "displayName":"Fred F. User",
									 "active":false
								},
								"updateAuthor":{
									 "self":"http://www.example.com/jira/rest/api/2/user?username=fred",
									 "name":"fred",
									 "displayName":"Fred F. User",
									 "active":false
								},
								"comment":"I did some work here.",
								"updated":"2016-03-16T04:22:37.471+0000",
								"visibility":{
									 "type":"group",
									 "value":"jira-developers"
								},
								"started":"2016-03-16T04:22:37.471+0000",
								"timeSpent":"3h 20m",
								"timeSpentSeconds":12000,
								"id":"100028",
								"issueId":"10002"
						 }
					]
			 },
			 "updated":"2016-04-06T02:36:53.594-0700",
			 "duedate":"2018-01-19",
			 "timetracking":{
					"originalEstimate":"10m",
					"remainingEstimate":"3m",
					"timeSpent":"6m",
					"originalEstimateSeconds":600,
					"remainingEstimateSeconds":200,
					"timeSpentSeconds":400
			 }
		},
		"names":{
			 "watcher":"watcher",
			 "attachment":"attachment",
			 "sub-tasks":"sub-tasks",
			 "description":"description",
			 "project":"project",
			 "comment":"comment",
			 "issuelinks":"issuelinks",
			 "worklog":"worklog",
			 "updated":"updated",
			 "timetracking":"timetracking"
		},
		"schema":{}
 }`
	tProjects = `
	[
		{
			 "expand":"description,lead,issueTypes,url,projectKeys,permissions,insight",
			 "self":"https://tmtest.atlassian.net/rest/api/3/project/10000",
			 "id":"10000",
			 "key":"IP",
			 "name":"ITSM project",
			 "avatarUrls":{
					"48x48":"https://tmtest.atlassian.net/secure/projectavatar?pid=10000&avatarId=10424",
					"24x24":"https://tmtest.atlassian.net/secure/projectavatar?size=small&s=small&pid=10000&avatarId=10424",
					"16x16":"https://tmtest.atlassian.net/secure/projectavatar?size=xsmall&s=xsmall&pid=10000&avatarId=10424",
					"32x32":"https://tmtest.atlassian.net/secure/projectavatar?size=medium&s=medium&pid=10000&avatarId=10424"
			 },
			 "projectTypeKey":"service_desk",
			 "simplified":false,
			 "style":"classic",
			 "isPrivate":false,
			 "properties":{

			 }
		}
 ]`
)

func TestJiraEvents(t *testing.T) {
	server := newJiraMockedServer()
	defer server.Close()

	testCases := map[string]struct {
		inType string
		inData string

		noPayload bool
		outType   string
	}{
		"create issue": {
			inType: v1alpha1.EventTypeJiraIssueCreate,
			inData: `{
				"fields": {
					 "project":
					 {
							"key": "EX-1"
					 },
					 "labels": ["alpha","beta"],
					 "description": "example bug report",
					 "issuetype": {
							"name": "Task"
					 },
					 "assignee": {
							"accountId": "5fe0704c9edf280075f188f0"
					 }
			 }
		}`,
			outType: v1alpha1.EventTypeJiraIssue,
		},

		"create issue - wrong payload": {
			inType: v1alpha1.EventTypeJiraIssueCreate,
			inData: `{"a":"b"}`,

			noPayload: true,
		},

		"get issue": {
			inType:  v1alpha1.EventTypeJiraIssueGet,
			inData:  `{"id":"` + tIssueID + `"}`,
			outType: v1alpha1.EventTypeJiraIssue,
		},

		"list projects": {
			inType:  v1alpha1.EventTypeJiraCustom,
			inData:  `{"method":"GET", "path":"/rest/api/3/project"}`,
			outType: v1alpha1.EventTypeJiraCustomResponse,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			// jira client configured to use test server
			jiraClient, err := jira.NewClient(nil, server.URL)
			if err != nil {
				assert.FailNow(t, "Could not build mocked Jira client: %v", err)
			}

			ceClient, send, responses := cetest.NewMockResponderClient(t, 1)

			ja := jiraAdapter{
				logger:   logtesting.TestLogger(t),
				ceClient: ceClient,

				jiraClient: jiraClient,
				baseURL:    server.URL,
				resSource:  tResSource,
			}

			go func() {
				if err := ja.Start(context.Background()); err != nil {
					assert.FailNow(t, "could not start test adapter")
				}
			}()

			in, err := createInEvent(tc.inType, tc.inData)
			require.Nil(t, err, "Could not create incoming event")

			send <- *in

			select {
			case event := <-responses:
				if tc.noPayload {
					assert.Nil(t, event.Event.Data(), "Unexpected event received")
					return
				}
				require.NotNil(t, event.Event.Data(), "Expected event was not received")

				assert.Equal(t, tc.outType, event.Event.Context.GetType())
				assert.Equal(t, tResSource, event.Event.Context.GetSource())

			case <-time.After(13 * time.Second):
				assert.Fail(t, "expected cloud event response was not received")
			}

		})
	}
}

func createInEvent(evType, evData string) (*cloudevents.Event, error) {
	data := make(map[string]interface{})

	if err := json.Unmarshal([]byte(evData), &data); err != nil {
		return nil, err
	}

	event := cloudevents.NewEvent()
	if err := event.SetData(tContentType, data); err != nil {
		return nil, err
	}

	event.SetID(tID)
	event.SetType(evType)
	event.SetSource(tSource)

	return &event, nil
}

func newJiraMockedServer() *httptest.Server {
	testMux := http.NewServeMux()
	testServer := httptest.NewServer(testMux)

	testMux.HandleFunc("/rest/api/2/issue", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			j := &jira.Issue{}
			err := json.NewDecoder(r.Body).Decode(j)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if j.Fields == nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, tIssue)
		default:
			w.WriteHeader(http.StatusBadRequest)
		}

	})

	testMux.HandleFunc("/rest/api/2/issue/"+tIssueID, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			fmt.Fprint(w, tIssue)
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	})

	testMux.HandleFunc("/rest/api/3/project", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			fmt.Fprint(w, tProjects)
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	})

	return testServer
}
