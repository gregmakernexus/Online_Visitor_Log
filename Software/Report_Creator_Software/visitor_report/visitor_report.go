package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	// "encoding/csv"
	"flag"
	"os"
	"path/filepath"
	"regexp"

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
	fmt.Println("Visitor Log Application V2.0")
	ctx := context.Background()
	flag.Parse()
	var err error
	log = debug.NewLogClient(*logLevel)
	// delete config file and re-input data
	if *clear {
		clearConfig()
	}
	c, err := readVisitorConfig(ctx, log)
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
		log.V(0).Fatalf("Unable to retrieve Sheets client: %v", err)
	}
	sheetID, err := s.GetSheetID(ctx, spreadsheetId, c.SheetName)
	where := ""
	switch {
	case err == nil: // temp sheet exists.
		/*-------------------------------------------------------------
		 * Read the sheet
		 *------------------------------------------------------------*/
		log.V(1).Printf("Sheet exists:%v id:%v\n", c.SheetName, sheetID)
		d, err := s.GetSheet(ctx, spreadsheetId, c.SheetName)
		if err != nil {
			log.V(0).Fatal(err)
		}
		/*--------------------------------------------------------------
		 * If there is data in the table, including the column headers
		 * get the last recNum written, otherwise read the whole table
		 *--------------------------------------------------------------*/
		if len(d) >= 2 {
			lastRecordNumber := c.getLastRecordNumber(ctx, d)
			where = "WHERE recNum > " + lastRecordNumber

		}
	case strings.Contains(err.Error(), "not found"):
		if sheetID, err = s.AddSheet(ctx, spreadsheetId, c.SheetName); err != nil {
			log.Fatalf("Error creating requested sheet. error:\n%v\n", err)
		}
		log.V(0).Printf("Added temp sheet:%v id:%v\n", c.SheetName, sheetID)
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
	/*------------------------------------------------------
	 *  Read the ovl_list table,  LastRecordNumber based recNum field of
	 *  the last record in the sheet
	 *-----------------------------------------------------*/
	log.V(0).Printf("Query: SELECT * FROM ovl_visits %v\n", where)
	r, err := db.Query("SELECT * FROM ovl_visits " + where)
	if err != nil {
		log.V(0).Fatal(err)
	}
	cols, err := r.Columns()
	if err != nil {
		fmt.Println("Failed to get columns", err)
		return
	}
	/*----------------------------------------------------------
	 * If previous sheet is empty then add the column headers
	 *---------------------------------------------------------*/
	update := make([][]string, 0)
	if where == "" {
		update = append(update, cols)
		fmt.Println(cols)
	}
	/*----------------------------------------------------------
	 * Convert the object returned from db to a 2d slice.
	 *---------------------------------------------------------*/
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
		update = append(update, result)
	}
	log.V(2).Printf("Appending %v records to %v\n", len(update), c.SheetName)
	for _, line := range update {
		log.V(2).Println(line)
	}
	/*---------------------------------------------------------------
	 * Write the slice to the google sheet named "Log"
	 *--------------------------------------------------------------*/
	if _, err = s.AppendSheet(ctx, spreadsheetId, c.SheetName, update); err != nil {
		log.Fatalf("AppendSheet error:%v", err)
	}

}

type visitor_config struct {
	DBName           string `json:"dbName"`
	URL              string `json:"url"`
	Userid           string `json:"id"`
	Pass             string `json:"pass"`
	SpreadSheetTitle string `json:"SpreadSheet"`
	SheetName        string `json:"Sheet"`
}

// Read the visitor_config file in the .makernexus directory.  Includes:
// 1. Create directories
// 2. If config does not exist, collect config info via cli.  Store to disk.
// 3. Read it back in.  Return *visitor_config
func readVisitorConfig(ctx context.Context, log *debug.DebugClient) (*visitor_config, error) {
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

// Write the visitor_config file in the .makernexus directory.  Update the LastRecordNumber:
func (c *visitor_config) getLastRecordNumber(ctx context.Context, data [][]string) string {
	if len(data) <= 2 {
		return "0"
	}
	end := len(data) - 1
	// Find the recNum Column
	columns := data[0]
	col := -1
	for i, column := range columns {
		if column == "recNum" {
			col = i
		}
	}
	if col == -1 {
		return "0"
	}
	// Store the LastRecordNumber.  Can't assume it matches the length of sheet
	// There is no consistency check.
	// Check if LastRecordNumber is numeric
	if !regexp.MustCompile(`\d`).MatchString(data[end][col]) {
		return "0"
	}
	return data[end][col]
}
