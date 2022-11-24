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

package googlesheettarget

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	logtesting "knative.dev/pkg/logging/testing"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

const sheetID = "1241k3jl1234hlk1234c1234ln125kch1"

const urlEscapedColonChar = "%3A"

func TestSheetHasAvailableRows(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	adapter := newAdapter(t)

	testCases := map[string]struct {
		response       sheets.ValueRange
		responseStatus int
		expectedResult bool
		expectedError  error
	}{
		"Sheet with less rows than maxSheetRow": {
			response:       sheets.ValueRange{},
			responseStatus: 200,
			expectedResult: true,
			expectedError:  nil,
		},
		"Sheet with more rows than maxSheetRow": {
			response:       sheets.ValueRange{Values: make([][]interface{}, maxSheetRow+1)},
			responseStatus: 200,
			expectedResult: false,
			expectedError:  nil,
		},
		"Sheet with number of rows equal to maxSheetRow": {
			response:       sheets.ValueRange{Values: make([][]interface{}, maxSheetRow)},
			responseStatus: 200,
			expectedResult: false,
			expectedError:  nil,
		},
		"Error response": {
			response:       sheets.ValueRange{},
			responseStatus: 500,
			expectedResult: false,
			expectedError: &googleapi.Error{
				Code:    500,
				Message: "",
				Body:    "{}",
				Header:  http.Header{"Content-Type": []string{"application/json"}},
				Errors:  []googleapi.ErrorItem(nil)},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			response, err := httpmock.NewJsonResponder(testCase.responseStatus, testCase.response)
			assert.NoError(t, err)

			mockURL := &url.URL{
				Path: fmt.Sprintf("/v4/spreadsheets/%s/values/!2:%d", sheetID, maxSheetRow+1),
			}
			// url.PathEscape() doesn't escape ':' in URL paths, but
			// the Google Sheet client does, which confuses httpmock.
			mockURLStr := strings.ReplaceAll(mockURL.String(), ":", urlEscapedColonChar)

			httpmock.RegisterResponder("GET", mockURLStr, response)

			result, err := adapter.sheetHasAvailableRows(&sheets.Sheet{Properties: &sheets.SheetProperties{}})

			if testCase.expectedError != nil {
				assert.EqualError(t, err, testCase.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, testCase.expectedResult, result)
		})
	}
}

func TestGetLatestSheetByName(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	adapter := newAdapter(t)

	const sheetsPrefix = "test"

	testCases := map[string]struct {
		response       sheets.Spreadsheet
		responseStatus int
		expectedSheet  *sheets.Sheet
		expectedError  interface{}
	}{
		"Sheet doesn't exists": {
			response:       sheets.Spreadsheet{},
			responseStatus: 200,
			expectedSheet:  nil,
			expectedError:  nil,
		},
		"Sheet exists": {
			response: sheets.Spreadsheet{Sheets: []*sheets.Sheet{{
				Properties: &sheets.SheetProperties{Title: "test 1"}},
			}},
			responseStatus: 200,
			expectedSheet:  &sheets.Sheet{Properties: &sheets.SheetProperties{Title: "test 1"}},
			expectedError:  nil,
		},
		"Last sheet is found": {
			response: sheets.Spreadsheet{Sheets: []*sheets.Sheet{
				{Properties: &sheets.SheetProperties{Title: "test 1"}},
				{Properties: &sheets.SheetProperties{Title: "test 2"}},
				{Properties: &sheets.SheetProperties{Title: "test 3"}},
			}},
			responseStatus: 200,
			expectedSheet:  &sheets.Sheet{Properties: &sheets.SheetProperties{Title: "test 3"}},
			expectedError:  nil,
		},
		"Sheets with different prefixes exists": {
			response: sheets.Spreadsheet{Sheets: []*sheets.Sheet{
				{Properties: &sheets.SheetProperties{Title: "first "}},
				{Properties: &sheets.SheetProperties{Title: "some un expected prefix"}},
				{Properties: &sheets.SheetProperties{Title: "test 1"}},
			}},
			responseStatus: 200,
			expectedSheet:  &sheets.Sheet{Properties: &sheets.SheetProperties{Title: "test 1"}},
			expectedError:  nil,
		},
		"Sheets without correct order": {
			response: sheets.Spreadsheet{Sheets: []*sheets.Sheet{
				{Properties: &sheets.SheetProperties{Title: "test 3"}},
				{Properties: &sheets.SheetProperties{Title: "test 2"}},
				{Properties: &sheets.SheetProperties{Title: "test 1"}},
			}},
			responseStatus: 200,
			expectedSheet:  &sheets.Sheet{Properties: &sheets.SheetProperties{Title: "test 3"}},
			expectedError:  nil,
		},
		"Sheets with expected prefix but without index exists": {
			response: sheets.Spreadsheet{Sheets: []*sheets.Sheet{
				{Properties: &sheets.SheetProperties{Title: "test 3"}},
				{Properties: &sheets.SheetProperties{Title: "test"}},
				{Properties: &sheets.SheetProperties{Title: "test 1"}},
			}},
			responseStatus: 200,
			expectedSheet:  &sheets.Sheet{Properties: &sheets.SheetProperties{Title: "test 3"}},
			expectedError:  nil,
		},
		"Error response": {
			response:       sheets.Spreadsheet{},
			responseStatus: 500,
			expectedSheet:  nil,
			expectedError: &googleapi.Error{
				Code: 500,
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			response, err := httpmock.NewJsonResponder(testCase.responseStatus, testCase.response)
			assert.NoError(t, err)
			mockURL := fmt.Sprintf("/v4/spreadsheets/%s", sheetID)
			httpmock.RegisterResponder("GET", mockURL, response)

			sheet, err := adapter.getLatestSheetByName(sheetsPrefix)

			if err != nil {
				assert.Equal(t, testCase.responseStatus, err.(*googleapi.Error).Code)
			}

			assert.Equal(t, testCase.expectedSheet, sheet)
		})
	}
}

func TestAppendValuesToSheet(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	adapter := newAdapter(t)

	someData := []string{"some", "data"}

	testCases := map[string]struct {
		response       sheets.BatchUpdateSpreadsheetResponse
		sheetToUpdate  *sheets.Sheet
		dataToInsert   []string
		responseStatus int
		expectedError  interface{}
	}{
		"Success row append": {
			response:       sheets.BatchUpdateSpreadsheetResponse{},
			sheetToUpdate:  &sheets.Sheet{Properties: &sheets.SheetProperties{SheetId: 123}},
			dataToInsert:   someData,
			responseStatus: 200,
			expectedError:  nil,
		},
		"Error appending without Properties": {
			response:       sheets.BatchUpdateSpreadsheetResponse{},
			sheetToUpdate:  &sheets.Sheet{},
			dataToInsert:   someData,
			responseStatus: 200,
			expectedError:  errors.New("sheet without SheetId can't be updated"),
		},
		"Error appending without SheetId": {
			response:       sheets.BatchUpdateSpreadsheetResponse{},
			sheetToUpdate:  &sheets.Sheet{Properties: &sheets.SheetProperties{SheetId: 0}},
			dataToInsert:   someData,
			responseStatus: 200,
			expectedError:  errors.New("sheet without SheetId can't be updated"),
		},
		"Error response": {
			response:       sheets.BatchUpdateSpreadsheetResponse{},
			sheetToUpdate:  &sheets.Sheet{Properties: &sheets.SheetProperties{SheetId: 123}},
			dataToInsert:   someData,
			responseStatus: 500,
			expectedError: &googleapi.Error{
				Code: 500,
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			response, err := httpmock.NewJsonResponder(testCase.responseStatus, testCase.response)
			assert.NoError(t, err)
			mockURL := fmt.Sprintf("/v4/spreadsheets/%s:batchUpdate", sheetID)
			httpmock.RegisterResponder("POST", mockURL, response)

			err = adapter.appendDataToSheet(testCase.sheetToUpdate, testCase.dataToInsert)
			if err != nil {
				if e, ok := err.(*googleapi.Error); ok {
					assert.Equal(t, testCase.responseStatus, e.Code)
				} else {
					assert.Equal(t, testCase.expectedError, err)
				}
			}

		})
	}
}

func TestCreateSheet(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	adapter := newAdapter(t)

	someData := []string{"some", "data"}

	testCases := map[string]struct {
		responseCreate       sheets.BatchUpdateSpreadsheetResponse
		responseAppend       sheets.BatchUpdateSpreadsheetResponse
		responseCreateStatus int
		responseAppendStatus int
		sheetName            string
		sheetNumber          int
		data                 []string
		expectedSheet        *sheets.Sheet
		expectedError        interface{}
	}{
		"Success sheet create": {
			responseCreate: sheets.BatchUpdateSpreadsheetResponse{
				UpdatedSpreadsheet: &sheets.Spreadsheet{
					Sheets: []*sheets.Sheet{
						{Properties: &sheets.SheetProperties{Title: "test 1", SheetId: 123}},
					},
				},
			},
			responseAppend:       sheets.BatchUpdateSpreadsheetResponse{},
			responseCreateStatus: 200,
			responseAppendStatus: 200,
			sheetName:            "test",
			sheetNumber:          1,
			data:                 someData,
			expectedSheet:        &sheets.Sheet{Properties: &sheets.SheetProperties{Title: "test 1", SheetId: 123}},
			expectedError:        nil,
		},
		"Error create sheet without sheetName": {
			responseCreate:       sheets.BatchUpdateSpreadsheetResponse{},
			responseAppend:       sheets.BatchUpdateSpreadsheetResponse{},
			responseCreateStatus: 200,
			responseAppendStatus: 200,
			sheetName:            "",
			sheetNumber:          1,
			data:                 someData,
			expectedSheet:        nil,
			expectedError:        errors.New("can't create sheet with empty name"),
		},
		"Error create sheet with negative number": {
			responseCreate:       sheets.BatchUpdateSpreadsheetResponse{},
			responseAppend:       sheets.BatchUpdateSpreadsheetResponse{},
			responseCreateStatus: 200,
			responseAppendStatus: 200,
			sheetName:            "test",
			sheetNumber:          -1,
			data:                 someData,
			expectedSheet:        nil,
			expectedError:        errors.New("can't create sheet with negative number"),
		},
		"Error create response": {
			responseCreate:       sheets.BatchUpdateSpreadsheetResponse{},
			responseAppend:       sheets.BatchUpdateSpreadsheetResponse{},
			responseCreateStatus: 500,
			responseAppendStatus: 200,
			sheetName:            "test",
			sheetNumber:          1,
			data:                 someData,
			expectedSheet:        nil,
			expectedError: &googleapi.Error{
				Code:    500,
				Message: "",
				Body:    "{}",
				Header:  http.Header{"Content-Type": []string{"application/json"}},
				Errors:  []googleapi.ErrorItem(nil)},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			mockURL := fmt.Sprintf("/v4/spreadsheets/%s:batchUpdate", sheetID)
			httpmock.RegisterResponder("POST", mockURL,
				// because of two http requests to the same endpoint need to check when and what to response
				func(request *http.Request) (*http.Response, error) {
					var requestData sheets.BatchUpdateSpreadsheetRequest
					err := json.NewDecoder(request.Body).Decode(&requestData)
					assert.NoError(t, err)

					// Create new sheet request
					if requestData.Requests[0].AddSheet != nil {
						response, err := httpmock.NewJsonResponse(testCase.responseCreateStatus, testCase.responseCreate)
						assert.NoError(t, err)
						return response, nil
					}
					// Append columns request
					if requestData.Requests[0].AppendCells != nil {
						response, err := httpmock.NewJsonResponse(testCase.responseAppendStatus, testCase.responseAppend)
						assert.NoError(t, err)
						return response, nil
					}
					// TODO: Fail testcase at this point, it is unexpected
					return httpmock.NewStringResponse(500, ``), nil
				},
			)

			sheet, err := adapter.createSheet(testCase.sheetName, testCase.sheetNumber)
			if err != nil {
				if e, ok := err.(*googleapi.Error); ok {
					assert.Equal(t, testCase.responseCreateStatus, e.Code)
				} else {
					assert.Equal(t, testCase.expectedError, err)
				}
			}

			assert.Equal(t, testCase.expectedSheet, sheet)
		})
	}
}

func TestGetIndexFromSheet(t *testing.T) {
	testCases := map[string]struct {
		sheet          sheets.Sheet
		expectedResult int
		expectedError  interface{}
	}{
		"Get sheet index success": {
			sheet:          sheets.Sheet{Properties: &sheets.SheetProperties{Title: "test 1"}},
			expectedResult: 1,
			expectedError:  nil,
		},
		"Error getting index with empty title": {
			sheet:          sheets.Sheet{Properties: &sheets.SheetProperties{Title: ""}},
			expectedResult: 0,
			expectedError:  errors.New("can't get index from sheet without Title"),
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {

			result, err := getIndexFromSheet(&testCase.sheet)
			assert.Equal(t, testCase.expectedError, err)

			assert.Equal(t, testCase.expectedResult, result)
		})
	}
}

func newAdapter(t *testing.T) *googleSheetAdapter {
	t.Helper()

	client, err := sheets.NewService(context.Background(), option.WithHTTPClient(http.DefaultClient))
	if err != nil {
		t.Fatal("Failed to create Google Sheet Service: " + err.Error())
	}

	return &googleSheetAdapter{
		client:  client,
		sheetID: sheetID,
		logger:  logtesting.TestLogger(t),
	}
}
