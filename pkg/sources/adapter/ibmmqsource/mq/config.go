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

// ConnectionConfig is IBM MQ common connection parameters.
type ConnectionConfig struct {
	QueueManager   string `envconfig:"QUEUE_MANAGER"`
	ChannelName    string `envconfig:"CHANNEL_NAME"`
	ConnectionName string `envconfig:"CONNECTION_NAME"`
	QueueName      string `envconfig:"QUEUE_NAME"`
}

// Delivery holds the delivery parameters used in the source.
type Delivery struct {
	DeadLetterQManager string `envconfig:"DEAD_LETTER_QUEUE_MANAGER"`
	DeadLetterQueue    string `envconfig:"DEAD_LETTER_QUEUE"`
	BackoffDelay       int    `envconfig:"BACKOFF_DELAY"`
	Retry              int    `envconfig:"DELIVERY_RETRY"`
}

// Auth contains IBM MQ authentication parameters.
type Auth struct {
	Username string `envconfig:"USER"`
	Password string `envconfig:"PASSWORD"`
	TLSConfig
}

// TLSConfig holds TLS connection parameters.
type TLSConfig struct {
	Cipher             string `envconfig:"TLS_CIPHER"`
	ClientAuthRequired bool   `envconfig:"TLS_CLIENT_AUTH"`
	CertLabel          string `envconfig:"TLS_CERT_LABEL"`
}
