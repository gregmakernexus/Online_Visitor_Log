package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	// "encoding/csv"
	"flag"
	"os"
	"path/filepath"
	"strings"

	"example.com/debug"
	"example.com/sheet"
	_ "github.com/go-sql-driver/mysql"
)

var logLevel = flag.Int("V", 0, "Logging level for debug messages")
var mimeType = flag.String("filter", "sheet", "Filter file type: sheet or document")
var clear = flag.Bool("clear", false, "Delete and reenter DB and sheet info")

// for supported mime types see: https://developers.google.com/drive/api/guides/mime-types
var mimetypeLookup = map[string]string{
	"sheet":    "application/vnd.google-apps.spreadsheet",
	"document": "application/vnd.google-apps.document",
}
var log *debug.DebugClient
var data [][]string

func main() {
	fmt.Println("Visitor Log Application V1.0")
	ctx := context.Background()
	flag.Parse()
	var err error
	log = debug.NewLogClient(*logLevel)
	// delete config file and re-input data
	if *clear {
		clearConfig()
	}
	c, err := initialize(ctx, log)
	if err != nil {
		log.V(0).Fatal(err)
	}
	o, err := sheet.NewOauth2Client(ctx, log)
	if err != nil {
		log.V(0).Fatal(err)
	}
	// Retrieve token from file, if not there generate a new token with Chrome/google
	if err = o.GetTokenFromFile("token.json"); err != nil {
		log.V(1).Printf("Error reading token from file. Error:%v\n", err)
		o.GetTokenFromWeb(ctx)
	}
	client := o.Oauth2Config.Client(ctx, o.Tok)

	/*---------------------------------------------------------------------------------------
	 * We are using a hardcoded sheet ID because it has been moved to the MakerNexus Shared Drive
	 *--------------------------------------------------------------------------------------*/
	spreadsheetId := "1sNn-hr3TbRXsZW6ACQqJyGO8woKxMVtmyNOlNUCtDAQ"
	c.SpreadSheetTitle = "Visitor Log"
	log.V(0).Printf("Updating SpreadSheet:%v id:%v\n", c.SpreadSheetTitle, spreadsheetId)
	/*-------------------------------------------------------------------------------------------
	 * Create the client structure for use below
	 *------------------------------------------------------------------------------------------*/
	s, err := sheet.NewSheetClient(ctx, log, client)
	if err != nil {
		log.V(0).Fatalf("sheet2csv Unable to retrieve Sheets client: %v", err)
	}
	/*---------------------------------------------------------------------
	 * Add a hidden sheet so we can delete without worry of being the last
	 * sheet.  Will get deleted if read is successful.
	 *-------------------------------------------------------------------*/
	s.AddSheet(ctx, spreadsheetId, "iwlwem987aad_9712")
	bogusID, _, _ := s.GetSheetID(ctx, spreadsheetId, "iwlwem987aad_9712")
	defer s.DeleteSheet(ctx, spreadsheetId, bogusID)
	/*---------------------------------------------------------------------
	 * Try to find the temporary sheet.  Make sure it is empty (delete/add)
	 *--------------------------------------------------------------------*/
	var tempName string = "temp_456123_" // made up name.
	var tempID int64
	tempID, _, err = s.GetSheetID(ctx, spreadsheetId, tempName)
	switch {
	case err == nil: // temp sheet exists.  delete and add to reset
		log.V(1).Printf("Aquired Sheet:%v id:%v\n", c.SheetName, tempID)
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
	 * Open the database.  Parameters were collected in the config
	 *--------------------------------------------------------------*/
	// @tcp(localhost:5555)/dbname?tls=skip-verify&autocommit=true
	dataSource := fmt.Sprintf("%v:%v@tcp(%v)/%v?tls=skip-verify",
		c.Userid, c.Pass, c.URL, c.DBName)
	log.V(0).Printf("db:%v\n", dataSource)
	db, err := sql.Open("mysql", dataSource)
	if err != nil {
		log.V(0).Fatal(err)
	}
	/*---------------------------------------------------------------
	 * Get list of tables in the database.  (not necessary for this app)
	 *--------------------------------------------------------------*/
	log.V(0).Printf("db open complete%v\n", db)
	r, err := db.Query("SHOW TABLES")
	if err != nil {
		log.V(0).Fatal(err)
	}
	var table string
	for r.Next() {
		r.Scan(&table)
		fmt.Println(table)
	}
	/*------------------------------------------------------
	 *  Read the ovl_list table.
	 *-----------------------------------------------------*/
	r, err = db.Query("SELECT * FROM ovl_visits")
	if err != nil {
		log.V(0).Fatal(err)
	}
	cols, err := r.Columns()
	if err != nil {
		fmt.Println("Failed to get columns", err)
		return
	}
	/*----------------------------------------------------------
	 * Convert the object returned from db to a 2d slice.
	 *---------------------------------------------------------*/
	data = make([][]string, 0)
	data = append(data, cols)
	fmt.Println(cols)
	// Result is your slice string.
	rawResult := make([][]byte, len(cols))
	dest := make([]interface{}, len(cols)) // A temporary interface{} slice
	for i := range rawResult {
		dest[i] = &rawResult[i] // Put pointers to each string in the interface slice
	}

	for r.Next() {
		err = r.Scan(dest...)
		if err != nil {
			fmt.Println("Failed to scan row", err)
			return
		}
		result := make([]string, len(cols))
		for i, raw := range rawResult {
			if raw == nil {
				result[i] = "\\N"
			} else {
				result[i] = string(raw)
			}
		}
		data = append(data, result)
	}
	log.V(2).Println("resulting 2d slice:")
	log.V(2).Println(data)
	/*---------------------------------------------------------------
	 * Write the slice to the google sheet named "temp_456123_"
	 *--------------------------------------------------------------*/
	if tempID, err = s.PutSheet(ctx, spreadsheetId, tempName, data); err != nil {
		log.Fatalf("PutSheet error:%v", err)
	}
	/*---------------------------------------------------------------
	 * See if the sheet exists.  Sheetname can be a subset of the total
	 * name, as long as it's unique.  If it exists delete it and re-rename.
	 *---------------------------------------------------------------*/
	var sheetID int64
	var actualSheetName string
	sheetID, actualSheetName, err = s.GetSheetID(ctx, spreadsheetId, c.SheetName)
	switch {
	case err == nil: // temp sheet exists.  delete and add to reset
		log.V(1).Printf("Aquired Sheet:%v id:%v\n", c.SheetName, sheetID)
		log.V(1).Printf("Deleting name:temp id:%v\n", sheetID)
		if err = s.DeleteSheet(ctx, spreadsheetId, sheetID); err != nil {
			log.Fatalf("Error deleting temp file. error:\n%v\n", err)
		}
	case strings.Contains(err.Error(), "not found"):
		actualSheetName = c.SheetName
	case strings.Contains(err.Error(), "multiple"):
		log.V(0).Fatalf("Multipe sheets include the temp name:%v.  Delete one.\n", tempName)
	}
	log.V(0).Printf("Re-naming temp sheet.  name:%v id:%v\n", actualSheetName, tempID)
	if err = s.RenameSheet(ctx, spreadsheetId, actualSheetName, tempID); err != nil {
		log.V(0).Fatalf("Error creating requested sheet. error:\n%v\n", err)
	}
	/*-------------------------------------------------------------------
	 * clean up the fake sheet we made so we can delete the last sheet
	 *-----------------------------------------------------------------*/
	s.DeleteSheet(ctx, spreadsheetId, bogusID)
}

type visitor_config struct {
	DBName           string `json:"dbName"`
	URL              string `json:"url"`
	Userid           string `json:"id"`
	Pass             string `json:"pass"`
	SpreadSheetTitle string `json:"SpreadSheet"`
	SheetName        string `json:"Sheet"`
}

// initialize the configuation information.  Includes:
// 1. Create directories
// 2. If config does not exist, collect config info via cli.  Store to disk.
// 3. Read it back in.  Return *visitor_config
func initialize(ctx context.Context, log *debug.DebugClient) (*visitor_config, error) {
	// Read config. if not there create it
	c := new(visitor_config)
	if err := c.dirSetup(); err != nil {
		return nil, err
	}
	_, err := os.Stat(".visitorConfig.json")
	if err != nil {
		log.V(0).Printf("Creating Configuration File")
		if err = c.build(".visitorConfig.json"); err != nil {
			log.V(0).Fatalf("Error creating config file. Error:%v", err)
		}
	}
	byteJson, err := os.ReadFile(".visitorConfig.json")
	if err != nil {
		log.V(0).Fatalf("Error reading config file")
	}
	err = json.Unmarshal(byteJson, c)
	return c, err
}

// clearConfig creates directories and deletes config file if it exists
func clearConfig() error {
	// Read config. if not there create it
	c := new(visitor_config)
	if err := c.dirSetup(); err != nil {
		return err
	}
	return os.Remove(".visitorConfig.json")

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
func (c *visitor_config) dirSetup() error {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	configPath := filepath.Join(home, ".makerNexus")
	if err := os.Chdir(configPath); err != nil {
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
	return nil
}

// Prompt user for config information and write it to disk in a hidden
// directory.
func (c *visitor_config) build(filename string) error {
	fmt.Println("Generating configuration file.  All fields are required. Hit ctrl-c to exit.")
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	configPath := filepath.Join(home, ".makerNexus")
	if err := os.Chdir(configPath); err != nil {
		return fmt.Errorf("error changing to home directory")
	}
	rd := bufio.NewScanner(os.Stdin)
	c.URL = input_config(rd, "Enter database URL (including port #):")
	c.DBName = input_config(rd, "Enter database name:")
	c.Userid = input_config(rd, "Enter database remote user: ")
	c.Pass = input_config(rd, "Enter database password: ")
	c.SpreadSheetTitle = input_config(rd, "Enter spreadsheet title: ")
	c.SheetName = input_config(rd, "Enter sheet name: ")
	buf, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filename, buf, 0777); err != nil {
		return err
	}
	return nil
}
