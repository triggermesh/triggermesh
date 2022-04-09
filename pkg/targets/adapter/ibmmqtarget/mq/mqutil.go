//go:build !noclibs

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

package mq

import (
	"strings"

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/ibmmqsource"
)

// IBM MQ target adapter constants.
const (
	CECorrelIDAttr          = "correlationid"
	KeyRepositoryExtensions = ".kdb"
)

// Object is a local wrapper for IBM MQ objects required to communicate with the queue.
type Object struct {
	queue *ibmmq.MQObject
	mqmd  *ibmmq.MQMD
	mqpmo *ibmmq.MQPMO
	mqgmo *ibmmq.MQGMO
	mqcbd *ibmmq.MQCBD
}

// NewConnection creates the connection to IBM MQ server.
func NewConnection(conn ConnectionConfig, auth Auth) (ibmmq.MQQueueManager, error) {
	// create IBM MQ channel definition
	channelDefinition := ibmmq.NewMQCD()
	channelDefinition.ChannelName = conn.ChannelName
	channelDefinition.ConnectionName = conn.ConnectionName

	// setup MQ connection params
	connOptions := ibmmq.NewMQCNO()
	connOptions.Options = ibmmq.MQCNO_CLIENT_BINDING
	connOptions.Options |= ibmmq.MQCNO_HANDLE_SHARE_BLOCK

	if auth.Cipher != "" {
		channelDefinition.SSLCipherSpec = auth.Cipher
		channelDefinition.SSLClientAuth = ibmmq.MQSCA_OPTIONAL
		if auth.ClientAuthRequired {
			channelDefinition.SSLClientAuth = ibmmq.MQSCA_REQUIRED
		}

		sco := ibmmq.NewMQSCO()
		sco.CertificateLabel = auth.CertLabel
		sco.KeyRepository = strings.TrimRight(ibmmqsource.KeystoreMountPath, KeyRepositoryExtensions)
		connOptions.SSLConfig = sco
	}
	connOptions.ClientConn = channelDefinition

	if auth.Username != "" {
		// init connection security params
		connSecParams := ibmmq.NewMQCSP()
		connSecParams.AuthenticationType = ibmmq.MQCSP_AUTH_USER_ID_AND_PWD
		connSecParams.UserId = auth.Username
		connSecParams.Password = auth.Password

		connOptions.SecurityParms = connSecParams
	}

	return ibmmq.Connx(conn.QueueManager, connOptions)
}

// OpenQueue opens IBM MQ queue.
func OpenQueue(queueName string, replyTo *ReplyTo, conn ibmmq.MQQueueManager) (Object, error) {
	// Create the Object Descriptor that allows us to give the queue name
	mqod := ibmmq.NewMQOD()
	mqod.ObjectType = ibmmq.MQOT_Q
	mqod.ObjectName = queueName

	qObject, err := conn.Open(mqod, ibmmq.MQOO_OUTPUT)
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
