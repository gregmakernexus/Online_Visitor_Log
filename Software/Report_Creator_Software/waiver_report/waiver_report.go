package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"example.com/debug"
	"example.com/sheet"
)

var logLevel = flag.Int("V", 0, "Logging level for debug messages")
var mimeType = flag.String("type", "sheet", "Google drive file type: sheet")
var spreadsheetTitle = flag.String("title", "", "Whole or partial sheet title. Must match one file only.")
var clear = flag.Bool("clear", false, "\"--clear=true\" to delete configuration and start over")
var sheetName = flag.String("sheet", "", "Sheet name to write(tab). If not exist create it.")
var csvName = flag.String("csv", "", "Absolute path to input CSV file.")

// for supported mime types see: https://developers.google.com/drive/api/guides/mime-types
var mimetypeLookup = map[string]string{
	"sheet":    "application/vnd.google-apps.spreadsheet",
	"document": "application/vnd.google-apps.document",
}
var log *debug.DebugClient

func main() {
	ctx := context.Background()
	flag.Parse()
	var err error
	log = debug.NewLogClient(*logLevel)
	if *clear {
		if err = clearConfig(); err != nil {
			log.V(0).Printf("Error deleting config file:%v\n", err)
			log.Fatal(-1)
		}
	}
	c, err := newSheetConfig(ctx, log, *spreadsheetTitle, *sheetName, *csvName)
	if err != nil {
		log.V(0).Fatal(err)
	}
	/*--------------------------------------------------------------------------------------
	 * Get oauth2 token from file or web
	 *-------------------------------------------------------------------------------------*/
	o, err := sheet.NewOauth2Client(ctx, log)
	if err != nil {
		log.V(0).Fatal(err)
	}
	// Retrieve token from file, if not there generate a new token with Chrome/google
	if err = o.GetTokenFromFile("token.json"); err != nil {
		o.GetTokenFromWeb(ctx)
	}
	client := o.Oauth2Config.Client(ctx, o.Tok)
	/*---------------------------------------------------------------------------------------
	 * Use the sheet name to obtain the spreadsheet id.
	 *--------------------------------------------------------------------------------------*/
	d, err := sheet.NewDriveClient(ctx, log, client)
	if err != nil {
		log.V(0).Fatal("err getting drive client")
	}

	sheetList, err := d.ListDir(ctx, mimetypeLookup[*mimeType], c.SpreadSheetTitle)
	if err != nil {
		log.V(0).Fatal(err)
	}
	switch {
	case len(sheetList) == 0:
		log.V(0).Fatalf("No sheets found.\n")
	case len(sheetList) > 1:
		log.V(0).Fatalf("Please specify a tighter sheet name match.  More than one found.\nfiles:%v\n",
			sheetList)
	}
	spreadsheetId := sheetList[0].Id
	*spreadsheetTitle = sheetList[0].Name
	log.V(0).Printf("Aquired SpreadSheet:%v id:%v\n", *spreadsheetTitle, spreadsheetId)
	/*-------------------------------------------------------------------------------------------
	 * Use sheet api to do things to operate on sheet
	 *------------------------------------------------------------------------------------------*/
	s, err := sheet.NewSheetClient(ctx, log, client)
	if err != nil {
		log.V(0).Fatalf("sheet2csv Unable to retrieve Sheets client: %v", err)
	}
	/*---------------------------------------------------------------------
	 * Add a hidden sheet so we can delete without worry of being the last
	 * sheet.
	 *-------------------------------------------------------------------*/
	s.AddSheet(ctx, spreadsheetId, "iwlwem987aad_9712")
	bogusID, _, _ := s.GetSheetID(ctx, spreadsheetId, "iwlwem987aad_9712")
	/*---------------------------------------------------------------------
	 * Try to find the temporary sheet.  Make sure it is empty (delete/add)
	 *--------------------------------------------------------------------*/
	var tempName string = "temp_456123_"
	var tempID int64
	tempID, _, err = s.GetSheetID(ctx, spreadsheetId, tempName)
	switch {
	case err == nil: // temp sheet exists.  delete and add to reset
		log.V(1).Printf("Aquired Sheet:%v id:%v\n", sheetName, tempID)
		if err = s.DeleteSheet(ctx, spreadsheetId, tempID); err != nil {
			log.Fatalf("Error deleting temp file. error:\n%v\n", err)
		}
		log.V(1).Printf("Deleted name:%v id:%v\n", tempName, tempID)
		if tempID, err = s.AddSheet(ctx, spreadsheetId, tempName); err != nil {
			log.V(0).Fatalf("Error creating requested sheet. error:%v.\n", err)
		}
		log.V(1).Printf("Re-added temp sheet:%v id:%v\n", tempName, tempID)
	case strings.Contains(err.Error(), "not found"):
		if tempID, err = s.AddSheet(ctx, spreadsheetId, tempName); err != nil {
			log.Fatalf("Error creating requested sheet. error:\n%v\n", err)
		}
		log.V(1).Printf("Added temp sheet:%v id:%v\n", tempName, tempID)
	case strings.Contains(err.Error(), "multiple"):
		log.V(0).Fatalf("Multipe sheets include the temp name:%v.  Delete one.\n", tempName)
	}
	/*--------------------------------------------------------------
	 * Read the csv into a slice
	 *--------------------------------------------------------------*/
	if c.CsvName == "" {
		log.V(0).Fatalf("CSV file name is missing")
	}
	log.V(0).Printf("Opening csv name:%v\n", c.CsvName)
	csvFile, err := os.Open(c.CsvName)
	if err != nil {
		log.V(0).Fatalf("failed open file: %v", err)
	}
	defer csvFile.Close()
	w := csv.NewReader(csvFile)
	data, err := w.ReadAll()
	if err != nil {
		log.Fatalf("error read csv error:%v\n", err)
	}
	log.V(0).Printf("CSV read complete file:%v\n", *csvName)
	/*---------------------------------------------------------------
	 * Write the slice to the google sheet named "temp"
	 *--------------------------------------------------------------*/
	if tempID, err = s.PutSheet(ctx, spreadsheetId, tempName, data); err != nil {
		log.Fatalf("PutSheet error:%v", err)
	}
	/*---------------------------------------------------------------
	 * See if the sheet exists.  Sheetname can be a subset of the total
	 * name, as long as it's unique.  If it exists delete it.
	 *---------------------------------------------------------------*/
	var sheetID int64
	var actualSheetName string
	sheetID, actualSheetName, err = s.GetSheetID(ctx, spreadsheetId, c.SheetName)
	switch {
	case err == nil: // temp sheet exists.  delete and add to reset
		log.V(1).Printf("Aquired Sheet:%v id:%v\n", sheetName, sheetID)
		log.V(1).Printf("Deleting name:temp id:%v\n", sheetID)
		if err = s.DeleteSheet(ctx, spreadsheetId, sheetID); err != nil {
			log.Fatalf("Error deleting temp file. error:\n%v\n", err)
		}
	case strings.Contains(err.Error(), "not found"):
		actualSheetName = c.SheetName
	case strings.Contains(err.Error(), "multiple"):
		log.V(0).Printf("Multipe sheets include the temp name:%v.  Delete one.\n", tempName)
		log.Fatal(-1)
	}
	log.V(0).Printf("Re-naming temp sheet.  name:%v id:%v\n", actualSheetName, tempID)
	if err = s.RenameSheet(ctx, spreadsheetId, actualSheetName, tempID); err != nil {
		log.V(0).Printf("Error creating requested sheet. error:\n%v\n", err)
		log.Fatal(-1)
	}
	// clean up the fake sheet we made so we can delete the last sheet
	s.DeleteSheet(ctx, spreadsheetId, bogusID)
	// Remove the directory path and filetype and rename the spreadhsst
	newName := filepath.Base(c.CsvName)
	newName = strings.Replace(newName, ".csv", "", -1)
	if err = s.ChangeTitle(ctx, spreadsheetId, newName); err != nil {
		log.V(0).Printf("Error changing spreadsheet title:%v\n", err)
		log.Fatal(-1)
	}
	log.V(0).Printf("Rename of spreadsheet complete.  New name is:%v\n", newName)
}

type sheet_config struct {
	CsvName          string `json:"CsvName"`
	SpreadSheetTitle string `json:"SpreadSheet"`
	SheetName        string `json:"Sheet"`
}
type waiver_config struct {
	U       string `json:"userid"`
	P       string `json:"password"`
	CsvName string `json:"file"`
}

// initialize the configuation information.  Includes:
// 1. Create directories
// 2. If config does not exist, collect config info via cli.  Store to disk.
// 3. Read it back in.  Return *visitor_config
func newSheetConfig(_ context.Context, log *debug.DebugClient,
	spreadsheetTitle, sheetName, csvName string) (*sheet_config, error) {
	// new sheet client
	c := new(sheet_config)
	// create .makerNexus directory if need and cd to that directory
	if err := c.dirSetup(); err != nil {
		return nil, err
	}
	// if config file does not exist then create it
	byteJson, err := os.ReadFile(".sheetConfig.json")
	if err != nil {
		log.V(0).Printf("Creating Configuration File")
		if err = c.build(".sheetConfig.json"); err != nil {
			log.V(0).Fatalf("Error creating config file. Error:%v", err)
		}
	}
	if err = json.Unmarshal(byteJson, c); err != nil {
		log.V(0).Printf("%v", err)
		log.Fatal(-1)
	}

	if spreadsheetTitle != "" {
		c.SpreadSheetTitle = spreadsheetTitle
	}
	if sheetName != "" {
		c.SheetName = sheetName
	}
	c.CsvName = csvName
	if csvName == "" {
		// get the csv name from the .waiversign.json config file
		w := new(waiver_config)
		if byteJson, err = os.ReadFile(".waiversign.json"); err != nil {
			log.V(0).Fatalf("waiver config file does not exist\n")
		}
		if err = json.Unmarshal(byteJson, w); err != nil {
			log.V(0).Fatalf("%v", err)
		}
		c.CsvName = w.CsvName
	}
	err = c.writeConfig(".sheetConfig.json")
	return c, err
}

// clearConfig creates directories and deletes config file if it exists
func clearConfig() error {
	// Read config. if not there create it
	c := new(sheet_config)
	if err := c.dirSetup(); err != nil {
		return err
	}
	return os.Remove(".sheetConfig.json")

}

// clearConfig creates directories and deletes config file if it exists
func (c *sheet_config) writeConfig(filename string) error {
	// create .makerNexus directory if need and cd to that directory
	if err := c.dirSetup(); err != nil {
		return err
	}
	// Read config. if not there create it
	buf, err := json.Marshal(c)
	if err != nil {
		return err
	}
	err = os.WriteFile(filename, buf, 0777)
	return err
}

// Cli read with prompt
func input_config(rd *bufio.Scanner, prompt string) string {
	done := false
	response := ""
	for x := 0; x < 10; x++ {
		if done {
			return response
		}
		fmt.Printf("%v", prompt)
		rd.Scan()
		response = rd.Text()
		if response != "" {
			done = true
		}
		time.Sleep(time.Second + 10)
	}
	return response
}

// Create directories
func (c *sheet_config) dirSetup() error {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	configPath := filepath.Join(home, ".makerNexus")
	if err := os.Chdir(home); err != nil {
		return fmt.Errorf("error changing to home directory")
	}
	/*----------------------------------------------------------------
	 * if directory does not exist then create it
	 *----------------------------------------------------------------*/
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.Mkdir(".makerNexus", 0777); err != nil {
			return fmt.Errorf("error creating directory .makerNexus")
		}
	}
	if err := os.Chdir(configPath); err != nil {
		return fmt.Errorf("error changing to home directory")
	}
	return nil
}

// Prompt user for config information and write it to disk in a hidden
// directory.
func (c *sheet_config) build(config_filename string) error {
	// create .makerNexus directory if need and cd to that directory
	if err := c.dirSetup(); err != nil {
		return err
	}
	fmt.Println("Generating configuration file.  All fields are required. Hit ctrl-c to exit.")
	rd := bufio.NewScanner(os.Stdin)
	if c.SpreadSheetTitle == "" {
		c.SpreadSheetTitle = input_config(rd, "Enter spreadsheet title: ")
	}
	if c.SheetName == "" {
		c.SheetName = input_config(rd, "Enter sheet name: ")
	}
	if c.CsvName == "" {
		c.CsvName = input_config(rd, "Enter CSV name (CR to read from waiversign config): ")
	}
	buf, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if err := os.WriteFile(config_filename, buf, 0777); err != nil {
		return err
	}
	return nil
}

func NewOauth2Client(ctx context.Context, log *debug.DebugClient) {
	panic("unimplemented")
}
