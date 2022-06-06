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
	"unicode"

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
	"github.com/triggermesh/triggermesh/pkg/sources/reconciler/ibmmqsource"
	"go.uber.org/zap"
)

// IBM MQ source adapter constants.
const (
	CECorrelIDAttr          = "correlationid"
	KeyRepositoryExtensions = ".kdb"
)

// Object is a local wrapper for IBM MQ objects required to communicate with the queue.
type Object struct {
	queue *ibmmq.MQObject
	dlq   *ibmmq.MQObject
	mqmd  *ibmmq.MQMD
	mqpmo *ibmmq.MQPMO
	mqgmo *ibmmq.MQGMO
	mqcbd *ibmmq.MQCBD
}

// Handler is a function used as IBM MQ callback.
type Handler func([]byte, string) error

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
func OpenQueue(queueName string, dlqName string, conn ibmmq.MQQueueManager) (Object, error) {
	mh, err := conn.CrtMH(ibmmq.NewMQCMHO())
	if err != nil {
		return Object{}, err
	}

	// Create the Object Descriptor that allows us to give the queue name
	mqod := ibmmq.NewMQOD()
	mqod.ObjectType = ibmmq.MQOT_Q
	mqod.ObjectName = queueName

	qObject, err := conn.Open(mqod, ibmmq.MQOO_OUTPUT|ibmmq.MQOO_INPUT_SHARED)
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

	pmo := ibmmq.NewMQPMO()
	pmo.Options = ibmmq.MQPMO_NO_SYNCPOINT

	putmqmd := ibmmq.NewMQMD()
	putmqmd.Format = ibmmq.MQFMT_STRING

	res := Object{
		queue: &qObject,
		mqmd:  putmqmd,
		mqgmo: gmo,
		mqpmo: pmo,
	}

	if dlqName != "" {
		mqod.ObjectName = dlqName
		dlqObject, err := conn.Open(mqod, ibmmq.MQOO_OUTPUT)
		if err != nil {
			return Object{}, err
		}
		res.dlq = &dlqObject
	}

	return res, nil
}

// RegisterCallback registers the callback function for the incoming messages in the target queue.
func (q *Object) RegisterCallback(f Handler, delivery Delivery, logger *zap.SugaredLogger) error {
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
			logger.Errorw("Callback received unexpected status", zap.Error(mqRet))
			return
		}
		cid := strings.TrimFunc(string(mqMD.CorrelId), func(r rune) bool {
			return !unicode.IsGraphic(r)
		})

		err := f(data, cid)
		if err != nil {
			logger.Errorw("Callback execution error", zap.Error(err))
			if mqMD.BackoutCount >= int32(delivery.Retry) {
				if delivery.DeadLetterQueue == "" {
					logger.Infof("Dead-letter queue is not set, discarding poisoned message %q", string(mqMD.MsgId))
				} else if err := q.sendToDLQ(data, mqMD); err != nil {
					logger.Errorw("Failed to forward the message to DLQ, discarding", zap.Error(err))
				}
				mqConn.Cmit()
				return
			}
			if err := mqConn.Back(); err != nil {
				logger.Errorw("Backout failed", zap.Error(err))
			}
			return
		}

		if err := mqConn.Cmit(); err != nil {
			logger.Errorf("Commit failed", zap.Error(err))
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
func (q *Object) PutToDLQ(mqmd *ibmmq.MQMD, data []byte) error {
	return q.dlq.Put(mqmd, q.mqpmo, data)
}

// Close closes the queue.
func (q *Object) Close() error {
	if q.dlq != nil {
		q.dlq.Close(0)
	}
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

func (q *Object) sendToDLQ(data []byte, mqmd *ibmmq.MQMD) error {
	// TODO: store handler error in dead letter header
	dlh := q.deadLetterHeader(*mqmd)
	return q.PutToDLQ(mqmd, append(dlh.Bytes(), data...))
}

// deadLetterHeader returns meta data for the poisoned message descriptor
func (q *Object) deadLetterHeader(mqmd ibmmq.MQMD) *ibmmq.MQDLH {
	dlh := ibmmq.NewMQDLH(&mqmd)
	dlh.Reason = ibmmq.MQRC_UNEXPECTED_ERROR
	dlh.DestQName = q.queue.Name
	dlh.PutApplType = ibmmq.MQAT_DEFAULT
	dlh.PutApplName = "TriggerMesh IBM MQ source adapter"
	return dlh
}
