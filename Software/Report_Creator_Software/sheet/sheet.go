package sheet

import (
	"context"
	"fmt"
	"net/http"

	"example.com/debug"
	"google.golang.org/api/option"
	sheets "google.golang.org/api/sheets/v4"
)

type SheetClient struct {
	srv *sheets.Service
}

var log *debug.DebugClient

func NewSheetClient(ctx context.Context, l *debug.DebugClient, client *http.Client) (*SheetClient, error) {
	var err error
	log = l
	c := new(SheetClient)
	c.srv, err = sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("NewSheetClient. error: %v", err)
	}
	return c, nil
}

// GetTitle returns sheet title.
func (c *SheetClient) GetTitle(ctx context.Context, spreadsheetID string) (string, error) {
	resp, err := c.srv.Spreadsheets.Get(spreadsheetID).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("unable to retrieve data from sheet: %v", err)
	}
	return resp.Properties.Title, nil
}

// GetTitle returns sheet title.
func (c *SheetClient) ChangeTitle(ctx context.Context, spreadsheetID string, spreadsheetTitle string) error {
	req := sheets.Request{
		UpdateSpreadsheetProperties: &sheets.UpdateSpreadsheetPropertiesRequest{
			Properties: &sheets.SpreadsheetProperties{
				Title: spreadsheetTitle,
			},
			Fields: "Title",
		},
	}
	rbb := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{&req},
	}
	_, err := c.srv.Spreadsheets.BatchUpdate(spreadsheetID, rbb).Context(context.Background()).Do()
	return err
}

// GetSheetNames returns sheet name list.
func (c *SheetClient) GetSheetID(ctx context.Context, spreadsheetID, sheetName string) (int64, error) {
	resp, err := c.srv.Spreadsheets.Get(spreadsheetID).Context(ctx).Do()
	if err != nil {
		return 0, fmt.Errorf("unable to retrieve data from spreadsheet: %v", err)
	}
	for i, s := range resp.Sheets {
		log.V(1).Printf("%v Sheet Name:%v ID:%v\n", i, s.Properties.Title, s.Properties.SheetId)
		if s.Properties.Title == sheetName {
			log.V(1).Printf("Found sheet:%v\n", sheetName)
			return s.Properties.SheetId, nil
		}
	}
	return 0, fmt.Errorf("sheet not found:%v", sheetName)
}

// GetSheet returns a Sheet in the form of a 2d slice.
func (c *SheetClient) GetSheet(ctx context.Context, spreadsheetID, sheetName string) ([][]string, error) {
	var data [][]string
	resp, err := c.srv.Spreadsheets.Values.Get(spreadsheetID, sheetName).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve data from sheet: %v", err)
	}
	if len(resp.Values) == 0 {
		return data, nil
	}
	var dataLength int = len(resp.Values[0])
	log.V(1).Println("header:")
	log.V(1).Println(resp.Values[0])
	log.V(1).Println("width of table:", dataLength)

	for _, r := range resp.Values {
		row := make([]string, dataLength)
		for j, cell := range r {
			row[j] = fmt.Sprintf("%v", cell)
		}
		data = append(data, row)
	}
	return data, nil
}

// PutSheet returns a Sheet.  Inputs a slice and write it to a sheet.
// Note: writing to the sheet will change the sheet id.
func (c *SheetClient) AppendSheet(ctx context.Context, spreadsheetID, sheetName string, putData [][]string) (sheetID int64, err error) {
	// The sheet api requires the data be loaded into a ValueRange structure.
	// ValueRange.Values is a interface{}, and can be loaded with append
	// var appendData *sheets.ValueRange
	appendData := new(sheets.ValueRange)
	for r := 0; r < len(putData); r++ {
		var row []interface{}
		for cols := 0; cols < len(putData[0]); cols++ {
			row = append(row, putData[r][cols])
		}
		appendData.Values = append(appendData.Values, row)
	}
	var appendResp *sheets.AppendValuesResponse
	log.V(1).Printf("Append spreadsheetID:%v range:%v\n", spreadsheetID, sheetName)
	appendResp, err = c.srv.Spreadsheets.Values.Append(spreadsheetID, sheetName, appendData).
		ValueInputOption("RAW").
		InsertDataOption("INSERT_ROWS").
		Context(ctx).
		Do()
	if err != nil {
		return 0, fmt.Errorf("append error: %v", err)
	}
	log.V(1).Printf("appendResp:%v\n", appendResp)
	sheetID, err = c.GetSheetID(ctx, spreadsheetID, sheetName)
	if err != nil {
		return 0, fmt.Errorf("missing sheet after append to %v", sheetName)
	}
	return sheetID, err
}

// Add a new sheet.
func (c *SheetClient) AddSheet(ctx context.Context, spreadsheetID, sheetName string) (int64, error) {
	req := sheets.Request{
		AddSheet: &sheets.AddSheetRequest{
			Properties: &sheets.SheetProperties{
				Title: sheetName,
			},
		},
	}
	rbb := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{&req},
	}
	_, err := c.srv.Spreadsheets.BatchUpdate(spreadsheetID, rbb).Context(context.Background()).Do()
	if err != nil {
		return 0, err
	}
	sheetID, err := c.GetSheetID(ctx, spreadsheetID, sheetName)
	if err != nil {
		return 0, fmt.Errorf("missing sheet after append to %v", sheetName)
	}
	return sheetID, nil
}

// Add a new hidden sheet.
func (c *SheetClient) AddHiddenSheet(ctx context.Context, spreadsheetID, sheetName string) (int64, error) {
	req := sheets.Request{
		AddSheet: &sheets.AddSheetRequest{
			Properties: &sheets.SheetProperties{
				Title:  sheetName,
				Hidden: true,
			},
		},
	}
	rbb := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{&req},
	}
	_, err := c.srv.Spreadsheets.BatchUpdate(spreadsheetID, rbb).Context(context.Background()).Do()
	if err != nil {
		return 0, err
	}
	sheetID, err := c.GetSheetID(ctx, spreadsheetID, sheetName)
	if err != nil {
		return 0, fmt.Errorf("missing sheet after append to %v", sheetName)
	}
	return sheetID, nil
}

// Delete a sheet by sheetID.
func (c *SheetClient) RenameSheet(ctx context.Context, spreadsheetID, newSheetName string, sheetID int64) error {
	req := sheets.Request{
		UpdateSheetProperties: &sheets.UpdateSheetPropertiesRequest{
			Properties: &sheets.SheetProperties{
				Title:   newSheetName,
				SheetId: sheetID,
			},
			Fields: "title",
		},
	}
	rbb := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{&req},
	}
	_, err := c.srv.Spreadsheets.BatchUpdate(spreadsheetID, rbb).Context(context.Background()).Do()
	if err != nil {
		return err
	}

	return nil
}

// Delete a sheet by sheetID.
func (c *SheetClient) DeleteSheet(ctx context.Context, spreadsheetID string, sheetID int64) error {
	req := sheets.Request{
		DeleteSheet: &sheets.DeleteSheetRequest{
			SheetId: sheetID,
		},
	}
	rbb := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{&req},
	}
	_, err := c.srv.Spreadsheets.BatchUpdate(spreadsheetID, rbb).Context(context.Background()).Do()
	if err != nil {
		return err
	}

	return nil
}
