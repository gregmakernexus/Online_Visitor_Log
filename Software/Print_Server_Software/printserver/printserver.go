package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	name "github.com/goombaio/namegenerator"
	"example.com/debug"
	clientinfo "example.com/clientinfo"
)

var dbURL = flag.String("db", "https://rfid.makernexuswiki.com/v1/OVLvisitorbadges.php", "Database Read URL")
var test = flag.Bool("test", false, "Label test to print labels with random names")
var printDelay = flag.Int("delay", 0, "Delay between print commands")
var logLevel = flag.Int("V", 0, "Logging level for debug messages")
var clearCache = flag.Bool("clear", false, "Clear database information cache")

// So we can have one set of code we have a set of filters to allow the following scenarios
//  1. --filter=printers.  Multiple printers for a large event.  We segment the alphabet and only print the specified range
//  2. --filter=camp.      Summer camp.
//  3. no filter specified This is the printer at the normal mod station.  If we are using #1 above.  This printer
//     must be power down.
var filterType = flag.String("filter", "", "Special filter. printers | camp")
var printRange = flag.String("range", "", "Multiple printers. Print labels where last name starts with. eg. a-f")
var campName = flag.String("name", "Kids Camp", "specify event name to print on badge")

var toMonth = []string{"", "Jan.", "Feb.", "Mar.", "Apr.", "Jun.", "Jul.", "Aug.", "Sep.", "Oct.", "Nov.", "Dec."}
var n name.Generator
var log *debug.DebugClient

type visitor map[string]any
type visitorList struct {
	Create string `json:"dateCreated"`
	Data   struct {
		Visitors []visitor `json:"visitors"`
	} `json:"data"`
}

type printer struct {
	labelTemplate       string
	printerModel        string
	printerManufacturer string
}
type labelClient struct {
	labelDir   string
	connection *http.Client
	printers   []printer
	current    int
}

func newLabelClient() *labelClient {
	c := new(labelClient)
	var labelByte []byte
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Error getting user home directory:%v\n", err)
	}
	// Read the label file into a string
	c.labelDir = filepath.Join(home, "Mylabels")
	if err := os.Chdir(c.labelDir); err != nil {
		log.Fatal("Label directory does not exist.")
	}
	// Find the CUPS printer
	out, err := exec.Command("lpstat", "-a").CombinedOutput()
	if err != nil {
		log.Fatalf("exec.Command() failed error:%v\n", err)
	}
	lines := strings.Split(string(out), "\n")
	if len(lines) == 0 {
		log.Fatalf("No Printers Found")
	}
	printers := make([]printer, 0)

	for _, line := range lines {
		tokens := strings.Split(line, " ")
		if len(tokens) < 2 {
			continue
		}
		var p printer
		p.printerModel = strings.ToUpper(tokens[0])
		switch {
		case strings.Contains(p.printerModel, "QL"):
			p.printerManufacturer = "BROTHER"
		case strings.Contains(p.printerModel, "DYMO"):
			p.printerManufacturer = "DYMO"
		default:
			continue
		} // switch
		fmt.Printf("Using a %v printer.  Model:%v\n", p.printerManufacturer, p.printerModel)
		labelByte, err = os.ReadFile(p.printerManufacturer + ".glabels")
		if err != nil {
			log.Fatalf("The label template is missing.  Please create template the program glabels_qt. \nError:%v", err)
		}
		p.labelTemplate = string(labelByte)
		printers = append(printers, p)
	} // for
	c.printers = printers
	c.current = 0
	// Create the http client with no security this is to access the database
	// website
	c.connection = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	return c
}

func (c *labelClient) urlRead(url string) ([]visitor, error) {
	// Do the http.get
	labels := new(visitorList)
	html, err := c.connection.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	// extract the body from the http packet
	results, err := io.ReadAll(html.Body)
	if err != nil {
		log.Fatal(err)
	}
	html.Body.Close()
	// unmarshall the json object
	if err := json.Unmarshal(results, labels); err != nil {
		return nil, err
	}
	return labels.Data.Visitors, nil

}
func (c *labelClient) print(info visitor) error {
	// to handle multiple printers, we will print to each printer in a
	// round robin fashion.  So get the printer info and get ready for next
	p := c.printers[c.current]
	c.current++
	if len(c.printers) >= c.current {
		c.current = 0
	}

	var temp string = p.labelTemplate
	// Get the date right now and update the label
	t := time.Now()
	nowDate := fmt.Sprintf("%v %v, %v", toMonth[t.Month()], t.Day(), t.Year())
	temp = strings.Replace(temp, "${Date}", nowDate, -1)
	reason := ""
	// loop through each field and update the label
	for key, data := range info {
		log.Println("key:", key, " data:", data)
		// do a special edit to make the reason for visit readable
		if key == "visitReason" {
			// The visitReason could be any or all of the radio buttons so we
			// need to check each of the possiblities
			switch {
			case strings.Contains(data.(string), "classOrworkshop"):
				reason = "Class/Workshop"
			case strings.Contains(data.(string), "makernexusevent"):
				reason = "Special Event"
			case strings.Contains(data.(string), "makernexuscamp"):
				reason = *campName
			case strings.Contains(data.(string), "volunteering"):
				reason = "Volunteer"
			case strings.Contains(data.(string), "forgotbadge"):
				reason = "Member"
			case strings.Contains(data.(string), "guest"),
				strings.Contains(data.(string), "tour"),
				strings.Contains(data.(string), "other"):
				reason = "Visitor"

			}
			temp = strings.Replace(temp, "Visitor", reason, -1)
			continue
		}
		dataType := fmt.Sprintf("%T", data)
		switch dataType {
		case "string":
			temp = strings.Replace(temp, "${"+key+"}", data.(string), -1)
		case "int":
			temp = strings.Replace(temp, "${"+key+"}", strconv.Itoa(data.(int)), -1)
		}
	}

	// cd to the Mylabel directory so we can write files
	if err := os.Chdir(c.labelDir); err != nil {
		log.Fatal("Label directory does not exist.")
	}
	// delete and write the temp.glables file
	os.Remove("temp.glabels")
	if err := os.WriteFile("temp.glabels", []byte(temp), 0666); err != nil {
		log.Fatalf("Error writing label file error:%v\n", err)
	}
	// print the label to the current printer
	printer := fmt.Sprintf("--printer %v", p)
	if out, err := exec.Command("glabels-batch-qt", printer, "temp.glabels").CombinedOutput(); err != nil {
		log.Fatalf("exec.Command failed error:%v\noutput:%v\n", err, out)
	}
	return nil
}

// Print a test page to each printer
func (c *labelClient) printTestPages() error {
	for _, p := range c.printers {
		var temp string = p.labelTemplate
		// Get the date right now and update the label
		t := time.Now()
		nowDate := fmt.Sprintf("%v %v, %v", toMonth[t.Month()], t.Day(), t.Year())
		temp = strings.Replace(temp, "${Date}", nowDate, -1)
		temp = strings.Replace(temp, "${nameFirst}", "WELCOME TO MAKERNEXUS", -1)
		temp = strings.Replace(temp, "${nameLast}", "", -1)
		temp = strings.Replace(temp, "Visitor", "Test Label", -1)
		// cd to the Mylabel directory so we can write files
		if err := os.Chdir(c.labelDir); err != nil {
			log.Fatal("Label directory does not exist.")
		}
		// delete and write the temp.glables file
		os.Remove("temp.glabels")
		if err := os.WriteFile("temp.glabels", []byte(temp), 0666); err != nil {
			log.Fatalf("Error writing label file error:%v\n", err)
		}
		// print the label
		printer := fmt.Sprintf("--printer %v", p)
		if out, err := exec.Command("glabels-batch-qt", printer, "temp.glabels").CombinedOutput(); err != nil {
			log.Fatalf("exec.Command failed error:%v\noutput:%v\n", err, out)
		}
	}
	return nil
}

func newFilter(filter string) (map[string]int, error) {
	alpha := "abcdefghijklmnopqrstuvwxyz"
	filter = strings.ReplaceAll(filter, " ", "")
	filterSplit := strings.Split(filter, "-")
	if len(filterSplit) != 2 && len(filterSplit[0]) != 1 && len(filterSplit[1]) != 1 {
		return nil, fmt.Errorf("empty or invalid filter:%v", filter)
	}
	var low, high int
	if low = strings.Index(alpha, strings.ToLower(filterSplit[0])); low == -1 {
		return nil, fmt.Errorf("low filter must be alpha:%v", filter)
	}
	if high = strings.Index(alpha, strings.ToLower(filterSplit[1])); high == -1 {
		return nil, fmt.Errorf("high filter must be alpha:%v", filter)
	}

	temp := min(low, high)
	high = max(low, high)
	low = temp
	filterMap := make(map[string]int)
	for i := low; i < high+1; i++ {
		filterMap[string(alpha[i])] = 0
	}
	return filterMap, nil
}

func okToPrint(filterMap map[string]int, label visitor) bool {
	// If not filtering, don't print camps
	switch {
	case isEmpty(*filterType): {
		for key, data := range label {
			log.Println("key:", key, " data:", data)
			// do a special edit to make the reason for visit readable
			if isEmpty(*filterType) && key == "visitReason" && strings.Contains(data.(string), "makernexuscamp") {
				fmt.Printf("skipping label because this printer is not a camp printer")
				return false
			}
		}
		return true
	}
	case *filterType == "camp":
	}	
	

	lastName := label["nameLast"].(string)
	firstChar := lastName[0:1]
	firstChar = strings.ToLower(firstChar)
	if _, exists := filterMap[firstChar]; !exists {
		fmt.Printf("skipping label.  lastName:%v filtermap:%v\n", lastName, filterMap)
		return false
	}
	return true
}

func main() {
	// init command line flags
	flag.Parse()
	
	
	var err error
	log = debug.NewLogClient(*logLevel)
	// delete config file and re-input data
	if *clearCache {
		clientinfo.clearConfig()
	}
	clients, err := clientinfo.newClientInfo(log)
	if err != nil {
		log.V(0).Fatal(err)
	}
	log.V(2).Println("resulting 2d slice:")
	for _,rec := range clients {
		log.V(2).Println(rec)
	}
	//  Create the label client
	c := newLabelClient()
	// Print program banners
	fmt.Println("Print Server v1.00.00  Initialized.  Hit control-c to exit.")
	// fmt.Println("Label Print Delay is:", *printDelay)
	// Print Test Page
	c.printTestPages()

	var labels []visitor
	// Generate the filter for the last name.
	var filterMap map[string]int
	if filterMap, err = newFilter(*printRange); err != nil {
		fmt.Printf("Error generating filter map:%v error:%v\n", filterMap, err)
		log.Fatal(3)
	}
	if *test {
		seed := time.Now().UTC().UnixNano()
		n = name.NewNameGenerator(seed)
	}
	for i := 1; ; i++ {
		switch {
		// if we are testing generate a fake name and process it
		case *test:
			randomName := n.Generate()
			n := strings.Split(randomName, "-")
			label := make(visitor)
			label["nameLast"] = n[1]
			label["nameFirst"] = n[0]
			label["URL"] = "https://makernexus.org;laskdfjas;ldkfjas;ldfkjas;ldfkjaasdf"
			labels = make([]visitor, 0)
			labels = append(labels, label)
		// This is a normal print server.  Get labels from web server
		case !*test && isEmpty(*filterType):
			if labels, err = c.urlRead(*dbURL); err != nil {
				return
			}
		// This is a summer camp printer
		case !*test && isEmpty(*printRange):
			// read database to see if there are labels to print
			if labels, err = c.urlRead(*dbURL); err != nil {
				return
			}

		} //switch
		// if there are no labels then print a dot and continue
		if len(labels) == 0 {
			// fmt.Printf("%v", ".")
			time.Sleep(time.Second)
			continue
		}
		// print all the labels return from database
		for _, label := range labels {
			// Filter label prints by last name

			if okToPrint(filterMap, label) {
				c.print(label)
			}
			if *printDelay == -1 {
				fmt.Printf("Hit enter to print next label>")
				fmt.Scanln()
			} else {
				fmt.Println("Label Print Delay is:", *printDelay)
				time.Sleep(time.Second * time.Duration(*printDelay))
			}
		} // for

	} // for infinite loop
}

func isEmpty(s string) bool {
	return len(s) == 0
}
