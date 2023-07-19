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
	"errors"
	"fmt"
	"strconv"
	"strings"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/metrics"
)

const maxSheetRow = 100

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: targets.GoogleSheetTargetResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	metrics.MustRegisterEventProcessingStatsView()

	env := envAcc.(*envAccessor)

	opts := make([]option.ClientOption, 0)
	if env.ServiceAccountKey != nil {
		opts = append(opts, option.WithCredentialsJSON(env.ServiceAccountKey))
	}

	sheetsService, err := sheets.NewService(ctx, opts...)
	if err != nil {
		logger.Fatalw("Error creating sheets client", zap.Error(err))
	}

	return &googleSheetAdapter{
		client:             sheetsService,
		sheetID:            env.SheetID,
		defaultSheetPrefix: env.DefaultSheetPrefix,

		ceClient: ceClient,
		logger:   logger,

		sr: metrics.MustNewEventProcessingStatsReporter(mt),
	}
}

var _ pkgadapter.Adapter = (*googleSheetAdapter)(nil)

type googleSheetAdapter struct {
	client             *sheets.Service
	sheetID            string
	defaultSheetPrefix string

	ceClient cloudevents.Client
	logger   *zap.SugaredLogger

	sr *metrics.EventProcessingStatsReporter
}

// Returns if stopCh is closed or Send() returns an error.
func (a *googleSheetAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting Google Sheet Adapter")

	if err := a.ceClient.StartReceiver(ctx, a.dispatch); err != nil {
		return err
	}

	return nil
}

func (a *googleSheetAdapter) dispatch(e cloudevents.Event) cloudevents.Result {
	var sheetName string
	var rows []string

	switch e.Type() {
	case v1alpha1.EventTypeGoogleSheetAppend:
		data := &SpreadsheetEvent{}
		if err := e.DataAs(data); err != nil {
			return fmt.Errorf("error processing incoming event data: %w", err)
		}

		sheetName = data.SheetName
		rows = data.Rows

	default:
		sheetName = a.defaultSheetPrefix
		rows = append(rows, e.String())
	}

	sheet, err := a.getOrCreateSheet(sheetName)
	if err != nil {
		return fmt.Errorf("error getting/creating sheet: %w", err)
	}

	if err := a.appendDataToSheet(sheet, rows); err != nil {
		return fmt.Errorf("error appending new values to sheet: %w", err)
	}

	a.logger.Debug("Successfully updated sheet")

	return cloudevents.ResultACK
}

func (a *googleSheetAdapter) appendDataToSheet(sheet *sheets.Sheet, rows []string) error {
	if sheet.Properties == nil || sheet.Properties.SheetId == 0 {
		return errors.New("sheet without SheetId can't be updated")
	}

	var valuesData []*sheets.CellData

	for i := range rows {
		valuesData = append(valuesData, &sheets.CellData{
			UserEnteredValue: &sheets.ExtendedValue{StringValue: &rows[i]},
		})
	}

	addValues := sheets.AppendCellsRequest{
		Fields:  "*",
		Rows:    []*sheets.RowData{{Values: valuesData}},
		SheetId: sheet.Properties.SheetId,
	}

	updateRequests := sheets.BatchUpdateSpreadsheetRequest{
		IncludeSpreadsheetInResponse: false,
		Requests:                     []*sheets.Request{{AppendCells: &addValues}},
		ResponseIncludeGridData:      false,
	}
	_, err := a.client.Spreadsheets.BatchUpdate(a.sheetID, &updateRequests).Do()
	if err != nil {
		return err
	}
	return nil
}

func (a *googleSheetAdapter) sheetHasAvailableRows(sheet *sheets.Sheet) (bool, error) {
	cellRange := fmt.Sprintf("%s!2:%d", sheet.Properties.Title, maxSheetRow+1)

	resp, err := a.client.Spreadsheets.Values.Get(a.sheetID, cellRange).Do()
	if err != nil {
		return false, err
	}
	// Empty trailing rows and columns are omitted and using that
	// we can just check number of rows in resp.Values
	if len(resp.Values) >= maxSheetRow {
		return false, nil
	}
	return true, nil
}

func (a *googleSheetAdapter) getLatestSheetByName(sheetName string) (*sheets.Sheet, error) {
	spreadSheet, err := a.client.Spreadsheets.Get(a.sheetID).Do()
	if err != nil {
		return nil, err
	}

	var lastSheet *sheets.Sheet
	lastSheetNumber := 0
	for _, sheet := range spreadSheet.Sheets {
		if strings.HasPrefix(sheet.Properties.Title, sheetName) {
			sheetNumber, err := getIndexFromSheet(sheet)
			if err != nil {
				continue
			}
			if lastSheetNumber <= sheetNumber {
				lastSheetNumber = sheetNumber
				lastSheet = sheet
			}
		}
	}
	return lastSheet, nil
}

func (a *googleSheetAdapter) createSheet(sheetName string, sheetNumber int) (*sheets.Sheet, error) {
	if sheetName == "" {
		return nil, errors.New("can't create sheet with empty name")
	}
	if sheetNumber < 0 {
		return nil, errors.New("can't create sheet with negative number")
	}

	lastWorksheetName := sheetName + " " + strconv.Itoa(sheetNumber)

	sheetAdd := sheets.AddSheetRequest{
		Properties: &sheets.SheetProperties{
			GridProperties: nil,
			Hidden:         false,
			SheetType:      "GRID",
			Title:          lastWorksheetName,
		},
	}

	updateRequests := sheets.BatchUpdateSpreadsheetRequest{
		IncludeSpreadsheetInResponse: true,
		Requests:                     []*sheets.Request{{AddSheet: &sheetAdd}},
		ResponseIncludeGridData:      false,
	}

	resp, err := a.client.Spreadsheets.BatchUpdate(a.sheetID, &updateRequests).Do()
	if err != nil {
		return nil, err
	}

	var newSheet *sheets.Sheet

	for _, sheet := range resp.UpdatedSpreadsheet.Sheets {
		if sheet.Properties.Title == lastWorksheetName {
			newSheet = sheet
			break
		}
	}

	if newSheet == nil {
		return nil, errors.New("sheet is not in the response")
	}

	return newSheet, nil
}

func (a *googleSheetAdapter) getOrCreateSheet(sheetName string) (*sheets.Sheet, error) {
	lastSheet, err := a.getLatestSheetByName(sheetName)
	if err != nil {
		return nil, err
	}
	if lastSheet == nil {
		return a.createSheet(sheetName, 1)
	}

	// if sheet does not exists or does not have empty rows in specified range
	available, err := a.sheetHasAvailableRows(lastSheet)
	if err != nil {
		return nil, err
	}
	if available {
		return lastSheet, nil
	}

	lastSheetNumber, err := getIndexFromSheet(lastSheet)
	if err != nil {
		return nil, err
	}

	return a.createSheet(sheetName, lastSheetNumber+1)
}

func getIndexFromSheet(sheet *sheets.Sheet) (int, error) {
	if sheet.Properties == nil || sheet.Properties.Title == "" {
		return 0, errors.New("can't get index from sheet without Title")
	}
	titleSplit := strings.Split(sheet.Properties.Title, " ")
	if len(titleSplit) == 2 {
		return strconv.Atoi(titleSplit[1])
	}
	return 0, errors.New("unexpected sheet Title format")
}
