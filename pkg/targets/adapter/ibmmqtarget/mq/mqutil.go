/*
Copyright 2021 TriggerMesh Inc.

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

package mq

import (
	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

// CECorrelIDAttr is the name of CloudEvent attribute used as IBM MQ Correlation ID.
const CECorrelIDAttr = "correlationid"

// Object is a local wrapper for IBM MQ objects required to communicate with the queue.
type Object struct {
	queue *ibmmq.MQObject
	mqmd  *ibmmq.MQMD
	mqpmo *ibmmq.MQPMO
	mqgmo *ibmmq.MQGMO
	mqcbd *ibmmq.MQCBD
}

// ConnConfig is a set of connection parameters.
type ConnConfig struct {
	ChannelName    string
	ConnectionName string
	User           string
	Password       string
	QueueManager   string
	QueueName      string
}

// ReplyTo holds the data used in MQ's Reply-to header.
type ReplyTo struct {
	Manager string
	Queue   string
}

// NewConnection creates the connection to IBM MQ server.
func NewConnection(cfg *ConnConfig) (ibmmq.MQQueueManager, error) {
	// create IBM MQ channel definition
	channelDefinition := ibmmq.NewMQCD()
	channelDefinition.ChannelName = cfg.ChannelName
	channelDefinition.ConnectionName = cfg.ConnectionName

	// init connection security params
	connSecParams := ibmmq.NewMQCSP()
	connSecParams.AuthenticationType = ibmmq.MQCSP_AUTH_USER_ID_AND_PWD
	connSecParams.UserId = cfg.User
	connSecParams.Password = cfg.Password

	// setup MQ connection params
	connOptions := ibmmq.NewMQCNO()
	connOptions.Options = ibmmq.MQCNO_CLIENT_BINDING
	connOptions.Options |= ibmmq.MQCNO_HANDLE_SHARE_BLOCK
	connOptions.ClientConn = channelDefinition
	connOptions.SecurityParms = connSecParams

	return ibmmq.Connx(cfg.QueueManager, connOptions)
}

// OpenQueue opens IBM MQ queue.
func OpenQueue(queueName string, replyTo *ReplyTo, conn ibmmq.MQQueueManager) (Object, error) {
	// Create the Object Descriptor that allows us to give the queue name
	mqod := ibmmq.NewMQOD()

	// We have to say how we are going to use this queue. In this case, to PUT
	// messages. That is done in the openOptions parameter.
	openOptions := ibmmq.MQOO_OUTPUT

	// Opening a QUEUE (rather than a Topic or other object type) and give the name
	mqod.ObjectType = ibmmq.MQOT_Q
	mqod.ObjectName = queueName

	qObject, err := conn.Open(mqod, openOptions)
	if err != nil {
		return Object{}, err
	}

	pmo := ibmmq.NewMQPMO()
	pmo.Options = ibmmq.MQPMO_NO_SYNCPOINT

	putmqmd := ibmmq.NewMQMD()
	putmqmd.Format = ibmmq.MQFMT_STRING
	putmqmd.ReplyToQMgr = replyTo.Manager
	putmqmd.ReplyToQ = replyTo.Queue

	return Object{
		queue: &qObject,
		mqmd:  putmqmd,
		mqpmo: pmo,
	}, nil
}

// Put puts the message to the queue.
func (q *Object) Put(data []byte, ceCorrelID string) error {
	correlID := [ibmmq.MQ_CORREL_ID_LENGTH]byte{}
	copy(correlID[:], ceCorrelID)
	mqmd := q.mqmd
	mqmd.CorrelId = correlID[:]
	return q.queue.Put(mqmd, q.mqpmo, data)
}

// Close closes the queue.
func (q *Object) Close() error {
	return q.queue.Close(0)
}
