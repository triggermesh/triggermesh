//go:build !codeanalysis

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
	"strings"
	"time"
	"unicode"

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
	"go.uber.org/zap"
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

// Delivery describes the message delivery details.
type Delivery struct {
	DeadLetterQManager string
	DeadLetterQueue    string
	BackoffDelay       int
	Retry              int
}

// Handler is a function used as IBM MQ callback.
type Handler func([]byte, string) error

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
func OpenQueue(queueName string, conn ibmmq.MQQueueManager) (Object, error) {
	mh, err := conn.CrtMH(ibmmq.NewMQCMHO())
	if err != nil {
		return Object{}, err
	}

	// Create the Object Descriptor that allows us to give the queue name
	mqod := ibmmq.NewMQOD()

	// We have to say how we are going to use this queue. In this case, to GET and PUT
	// messages. That is done in the openOptions parameter.
	openOptions := ibmmq.MQOO_OUTPUT

	// Opening a QUEUE (rather than a Topic or other object type) and give the name
	mqod.ObjectType = ibmmq.MQOT_Q
	mqod.ObjectName = queueName

	qObject, err := conn.Open(mqod, openOptions)
	if err != nil {
		return Object{}, err
	}

	// The GET/MQCB requires control structures, the Message Descriptor (MQMD)
	// and Get Options (MQGMO). Create those with default values.
	gmo := ibmmq.NewMQGMO()
	gmo.Options = ibmmq.MQGMO_SYNCPOINT

	// Set options to wait for a maximum of 3 seconds for any new message to arrive
	gmo.Options |= ibmmq.MQGMO_WAIT
	gmo.WaitInterval = 3 * 1000 // The WaitInterval is in milliseconds

	gmo.Options |= ibmmq.MQGMO_PROPERTIES_IN_HANDLE
	gmo.MsgHandle = mh

	return Object{
		queue: &qObject,
		mqmd:  ibmmq.NewMQMD(),
		mqgmo: gmo,
	}, nil
}

// RegisterCallback registers the callback function for the incoming messages in the target queue.
func (q *Object) RegisterCallback(f Handler, delivery *Delivery, log *zap.SugaredLogger) error {
	handler := func(
		mqConn *ibmmq.MQQueueManager,
		mqObj *ibmmq.MQObject,
		mqMD *ibmmq.MQMD,
		mqGMO *ibmmq.MQGMO,
		data []byte,
		mqCBC *ibmmq.MQCBC,
		mqRet *ibmmq.MQReturn,
	) {
		if mqRet.MQRC == ibmmq.MQRC_NO_MSG_AVAILABLE {
			return
		}
		if mqRet.MQCC != ibmmq.MQCC_OK {
			log.Errorf("Callback received unexpected status: %s", mqRet.Error())
			return
		}
		cid := strings.TrimFunc(string(mqMD.CorrelId), func(r rune) bool {
			return !unicode.IsGraphic(r)
		})

		err := f(data, cid)
		if err != nil {
			log.Errorf("Callback execution error: %v", err)
			if mqMD.BackoutCount >= int32(delivery.Retry) {
				if err := q.sendToDLQ(data, delivery); err != nil {
					log.Errorf("Failed to forward the message to DLQ: %v", err)
				}
				mqConn.Cmit()
				return
			}
			if err := mqConn.Back(); err != nil {
				log.Errorf("Backout failed: %v", err)
			}
			return
		}

		if err := mqConn.Cmit(); err != nil {
			log.Errorf("Commit failed: %v", err)
		}
	}

	// The MQCBD structure is used to specify the function to be invoked
	// when a message arrives on a queue
	q.mqcbd = ibmmq.NewMQCBD()
	q.mqcbd.CallbackFunction = handler

	// Register the callback function along with any selection criteria from the
	// MQMD and MQGMO parameters
	return q.queue.CB(ibmmq.MQOP_REGISTER, q.mqcbd, q.mqmd, q.mqgmo)
}

// StartListen sends the signal to IBM MQ server to start callback invocation.
func (q *Object) StartListen(conn ibmmq.MQQueueManager) error {
	// Then we are ready to enable the callback function. Any messages
	// on the queue will be sent to the callback
	ctlo := ibmmq.NewMQCTLO() // Default parameters are OK
	return conn.Ctl(ibmmq.MQOP_START, ctlo)
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

// Deallocate the message handle
func (q *Object) DeleteMessageHandle() error {
	return q.mqgmo.MsgHandle.DltMH(ibmmq.NewMQDMHO())
}

// Deregister the callback function - have to do this before the message handle can be
// successfully deleted
func (q *Object) DeregisterCallback() error {
	return q.queue.CB(ibmmq.MQOP_DEREGISTER, q.mqcbd, q.mqmd, q.mqgmo)
}

// Stop the callback function from being called again
func (q *Object) StopCallback(conn ibmmq.MQQueueManager) error {
	return conn.Ctl(ibmmq.MQOP_STOP, ibmmq.NewMQCTLO())
}

func (q *Object) sendToDLQ(data []byte, delivery *Delivery) error {
	// TODO: store handler error in dead letter header
	dlh := q.deadLetterHeader(ibmmq.MQRC_OPERATION_ERROR, delivery)
	data = append(dlh.Bytes(), data...)
	return q.Put(data, "")
}

// deadLetterHeader returns meta data for the poisoned message descriptor
func (q *Object) deadLetterHeader(reason int32, delivery *Delivery) *ibmmq.MQDLH {
	timestamp := time.Now()

	dlh := ibmmq.NewMQDLH(q.mqmd)
	dlh.Reason = reason
	dlh.DestQName = delivery.DeadLetterQueue
	dlh.DestQMgrName = delivery.DeadLetterQManager
	dlh.PutTime = timestamp.Format("030405")
	dlh.PutDate = timestamp.Format("20060102")

	return dlh
}
