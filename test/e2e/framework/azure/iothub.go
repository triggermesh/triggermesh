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

package azure

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/iothub/armiothub"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
)

// CreateIOTHubComponents Will create the Azure IOT Hub, and a device to produce data
func CreateIOTHubComponents(ctx context.Context, subscriptionID, rg, region, name string) (string, string) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		framework.FailfWithOffset(2, "Unable to authenticate: %s", err)
	}

	// Create the new iothub
	iothubClient := armiothub.NewResourceClient(subscriptionID, cred, nil)

	hub, err := iothubClient.BeginCreateOrUpdate(ctx, rg, name, armiothub.Description{
		Location: &region,
		Tags:     map[string]*string{E2EInstanceTagKey: to.StringPtr(name)},
		SKU: &armiothub.SKUInfo{
			Name:     armiothub.IotHubSKUF1.ToPtr(),
			Capacity: to.Int64Ptr(1),
			Tier:     armiothub.IotHubSKUTierFree.ToPtr(),
		},
		Identity: &armiothub.ArmIdentity{
			Type: armiothub.ResourceIdentityTypeNone.ToPtr(),
		},
	}, nil)
	if err != nil {
		framework.FailfWithOffset(2, "Unable to create iothub: %s", err)
	}

	_, err = hub.PollUntilDone(ctx, time.Second*30)
	if err != nil {
		framework.FailfWithOffset(2, "Unable to create iothub: %s", err)
	}

	resp, err := iothubClient.GetKeysForKeyName(ctx, rg, name, "iothubowner", nil)
	if err != nil {
		framework.FailfWithOffset(2, "Unable to get iothubowner key: %s", err)
	}

	devKey, err := CreateDevice(name, "testdev", *resp.PrimaryKey)
	if err != nil {
		framework.FailfWithOffset(2, "Unable to create iothub device: %s", err)
	}

	return devKey, fmt.Sprintf("HostName=%s.azure-devices.net;SharedAccessKeyName=iothubowner;SharedAccessKey=%s", name, *resp.PrimaryKey)
}

// CreateDevice relies on the REST API to create a new device for testing returning the primary key used for the newly created device.
func CreateDevice(hubName, deviceName, iotownerKey string) (string, error) {
	payload := fmt.Sprintf("{\"deviceId\":\"%s\",\"status\":\"enabled\",\"capabilities\":{\"iotEdge\":false},\"authentication\":{\"type\":\"sas\",\"symmetricKey\":{\"primaryKey\":null,\"secondaryKey\":null},\"x509Thumbprint\":null},\"deviceScope\":\"\",\"parentScopes\":[]}", deviceName)
	baseURL := fmt.Sprintf("%s.azure-devices.net/devices/%s", hubName, deviceName)

	nr := strings.NewReader(payload)

	req, _ := http.NewRequest(http.MethodPut, "https://"+baseURL+"?api-version=2020-03-13", nr)
	req.Header.Add("Authorization", CreateSaSToken(baseURL, "iothubowner", iotownerKey, true))
	req.Header.Add("content-type", "application/json")

	resp, _ := http.DefaultClient.Do(req)
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("failed to create device: %s", buf.String())
	}

	resPayload := make(map[string]interface{})
	_ = json.Unmarshal(buf.Bytes(), &resPayload)

	auth := resPayload["authentication"]
	keyList := auth.(map[string]interface{})["symmetricKey"]
	key := keyList.(map[string]interface{})["primaryKey"].(string)

	return key, nil
}

// CreateSaSToken will create a shared access signature to interact with the Azure
// IOTEventHub for device management.
func CreateSaSToken(uri, name, key string, withName bool) string {
	u := url.QueryEscape(uri)
	expirationTime := time.Now().Add(time.Minute * 15).Unix()
	decodedKey, _ := base64.StdEncoding.DecodeString(key)

	str2sign := u + "\n" + fmt.Sprintf("%v", expirationTime)

	h := hmac.New(sha256.New, decodedKey)
	h.Write([]byte(str2sign))
	sha := url.QueryEscape(base64.StdEncoding.EncodeToString(h.Sum(nil)))

	s := fmt.Sprintf("SharedAccessSignature sr=%s&sig=%s&se=%v", u, sha, expirationTime)

	if withName {
		s += "&skn=" + name
	}

	return s
}
