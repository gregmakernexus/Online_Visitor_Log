// label takes the data form OVL database and exports it to .glabels file
package label

import (
	"crypto/tls"
	"encoding/json"

	//"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"example.com/debug"
	client "example.com/clientinfo"
)

// So we can have one set of code we have a set of filters to allow the following scenarios
//  1. --filter=printers.  Multiple printers for a large event.  We segment the alphabet and only print the specified range
//  2. --filter=camp.      Summer camp.
//  3. no filter specified This is the printer at the normal mod station.  If we are using #1 above.  This printer
//     must be power down.
var toMonth = []string{"", "Jan.", "Feb.", "Mar.", "Apr.", "May", "Jun.", "Jul.", "Aug.", "Sep.", "Oct.", "Nov.", "Dec."}
var log *debug.DebugClient

type Visitor map[string]any
type VisitorList struct {
	Create string `json:"dateCreated"`
	Data   struct {
		Visitors []Visitor `json:"visitors"`
	} `json:"data"`
}

type Printer struct {
	LabelTemplate       string
	PrinterModel        string
	PrinterManufacturer string
}
type LabelClient struct {
	LabelDir   string
	connection *http.Client
	Printers   []Printer
	Current    int
	Log        *debug.DebugClient
}

var clients map[string][]string

func NewLabelClient(log *debug.DebugClient) *LabelClient {
	log.V(1).Printf("NewLabelClient started\n")
	c := new(LabelClient)
	c.Log = log
	var labelByte []byte
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Error getting user home directory:%v\n", err)
	}
	// Read the label file into a string
	c.LabelDir = filepath.Join(home, "Mylabels")
	if err := os.Chdir(c.LabelDir); err != nil {
		log.V(0).Printf("Label directory does not exist. path:%v\n", c.LabelDir)
		return nil
	}
	// scan for the label printer in usb ports (brother or dymo)
	out, err := exec.Command("lsusb").CombinedOutput()
	log.V(2).Printf("lsusb exec.command error:%v\n", string(out))
	if err != nil {
		log.Fatalf("lsusb exec.Command() failed error:%v\n", err)
	}
	buf := string(out)
	var p Printer
	switch {
	case strings.Contains(buf, "BROTHER"):
		p.PrinterManufacturer = "BROTHER"
	case strings.Contains(buf, "DYMO"):
		p.PrinterManufacturer = "DYMO"
	}
	log.V(2).Printf("Printer Manufacturer:%v\n", p.PrinterManufacturer)
	// Find out if cups driver is installed for glabels.  lpstat will
	// show the printer as ready even if it is not plugged in.
	out, err = exec.Command("lpstat", "-a").CombinedOutput()
	if err != nil {
		log.Fatalf("exec.Command() failed error:%v\n", err)
	}
	lines := strings.Split(string(out), "\n")
	if len(lines) == 0 {
		log.Fatalf("No Printers Found")
	}
	printers := make([]Printer, 0)
	i := 0
	for _, line := range lines {
		log.V(2).Printf("lpstat:%v\n", line)
		tokens := strings.Split(line, " ")
		if len(tokens) < 2 {
			continue
		}
		switch {
		case p.PrinterManufacturer == "BROTHER" && strings.Contains(tokens[0], "QL"),
			p.PrinterManufacturer == "DYMO" && strings.Contains(tokens[0], "LabelWriter"):
			p.PrinterModel = tokens[0]
			c.Current = i
			labelByte, err = os.ReadFile(p.PrinterManufacturer + ".glabels")
			if err != nil {
				log.Fatalf("The label template is missing.  Please create template the program glabels_qt. \nError:%v", err)
			}
			p.LabelTemplate = string(labelByte)
			log.V(2).Printf("add printer [%v:%v]\n", p.PrinterManufacturer,p.PrinterModel)
			printers = append(printers, p)
			i++
		}
	} // for
	c.Printers = printers
	
	// Create the http client with no security this is to access the OVL database
	// website
	c.connection = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	// load the clientinfo table into map for lookup
	clients, err = client.NewClientInfo(log)
	if err != nil {
		log.V(0).Fatal(err)
	}
	log.V(2).Println("client map:")
	for key, rec := range clients {
		log.V(2).Println(key, rec)
	}
	
	log.V(1).Printf("Return NewLabelClient\n")
	return c
}

func (c *LabelClient) ReadOVL(url string) ([]Visitor, error) {
	// Do the http.get
	log := c.Log
	labels := new(VisitorList)
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
func (c *LabelClient) ExportToGlabels(info Visitor) (Printer, error) {
	log := c.Log
	/*---------------------------------------------------------------
	* If there is multiple printers, choose printer in a round robin
	* fashion
	*---------------------------------------------------------------*/ 
	p := c.Printers[c.Current]
	var temp string = c.Printers[c.Current].LabelTemplate
	// Get the date right now and update the label
	t := time.Now()
	nowDate := fmt.Sprintf("%v %v, %v", toMonth[t.Month()], t.Day(), t.Year())
	temp = strings.Replace(temp, "${Date}", nowDate, -1)
	// Find the first and last name
	var first, last, reason string
	for key, data := range info {
		log.V(0).Printf("key:%v\n",key)
		switch key {
		case "nameFirst":
		    log.V(1).Printf("found first:%v\n",data.(string))
			temp = strings.Replace(temp, "${nameFirst}", data.(string), -1)
			first = data.(string)
		case "nameLast":
			log.V(1).Printf("found last:%v\n",data.(string))
			temp = strings.Replace(temp, "${nameLast}", data.(string), -1)
			last = data.(string)
		case "visitReason":
			log.V(1).Printf("visitReason:%v\n",data.(string))
			switch {
			case strings.Contains(data.(string), "classOrworkshop"):
				reason = "Class/Workshop"
			case strings.Contains(data.(string), "makernexusevent"):
				reason = "Special Event"
			case strings.Contains(data.(string), "makernexuscamp"):
				reason = "Kids Camp"
			case strings.Contains(data.(string), "volunteering"):
				reason = "Volunteer"
			case strings.Contains(data.(string), "forgotbadge"):
				reason = "Forgot Badge"
			case strings.Contains(data.(string), "guest"),
				strings.Contains(data.(string), "tour"),
				strings.Contains(data.(string), "other"):
				reason = "Visitor"
			}
			log.V(1).Printf("Reason:%v\n",reason)

		default:
			dataType := fmt.Sprintf("%T", data)
			switch dataType {
			case "string":
				temp = strings.Replace(temp, "${"+key+"}", data.(string), -1)
			case "int":
				temp = strings.Replace(temp, "${"+key+"}", strconv.Itoa(data.(int)), -1)
			}
		}
	}
	// We have scanned the entire OVL record.  Now do special lookup
	// when the person forgot a badge.  Check if member or staff
	if reason == "Forgot Badge" {
		key := strings.ToLower(last + first)
		log.V(1).Printf("forgot badge lookup:%v\n",key)
		reason = "Visitor"
		if c, exist := clients[key]; exist {
			log.V(1).Printf("lookup was found:%v length:%v\n",c,len(c))
			for i,t := range c {
			  log.V(1).Printf("clientinfo %v:%v\n",i,t)	
			}
			reason = "Member"
			s := strings.ToLower(c[7])
			if strings.Contains(s, "staff") {
				log.V(1).Printf("is a staff member")
				reason = "Staff"
			}
		}
	}
	temp = strings.Replace(temp, "Visitor", reason, -1)

	// cd to the Mylabel directory so we can write files
	if err := os.Chdir(c.LabelDir); err != nil {
		return p, fmt.Errorf("%v directory does not exist",c.LabelDir)
	}
	// delete and write the temp.glables file
	os.Remove("temp.glabels")
	if err := os.WriteFile("temp.glabels", []byte(temp), 0666); err != nil {
		return p, fmt.Errorf("write to temp.glabels error:%v\n", err)
	}
	// Successful export to file.  Bump to the next printer
	c.Current++
	if len(c.Printers) >= c.Current {
		c.Current = 0
	}
	return p, nil
}

// Print a test page to each printer
func (c *LabelClient) ExportTestToGlabels() error {
	for _, p := range c.Printers {
		var temp string = p.LabelTemplate
		// Get the date right now and update the label
		t := time.Now()
		nowDate := fmt.Sprintf("%v %v, %v", toMonth[t.Month()], t.Day(), t.Year())
		temp = strings.Replace(temp, "${Date}", nowDate, -1)
		temp = strings.Replace(temp, "${nameFirst}", "WELCOME TO MAKERNEXUS", -1)
		temp = strings.Replace(temp, "${nameLast}", "", -1)
		temp = strings.Replace(temp, "Visitor", "Test Label", -1)
		// cd to the Mylabel directory so we can write files
		if err := os.Chdir(c.LabelDir); err != nil {
			log.Fatal("Label directory does not exist.")
		}
		// delete and write the temp.glables file
		os.Remove("temp.glabels")
		if err := os.WriteFile("temp.glabels", []byte(temp), 0666); err != nil {
			log.Fatalf("Error writing label file error:%v\n", err)
		}

	}
	return nil
}

// func newFilter(filter string) (map[string]int, error) {
// 	alpha := "abcdefghijklmnopqrstuvwxyz"
// 	filter = strings.ReplaceAll(filter, " ", "")
// 	filterSplit := strings.Split(filter, "-")
// 	if len(filterSplit) != 2 && len(filterSplit[0]) != 1 && len(filterSplit[1]) != 1 {
// 		return nil, fmt.Errorf("empty or invalid filter:%v", filter)
// 	}
// 	var low, high int
// 	if low = strings.Index(alpha, strings.ToLower(filterSplit[0])); low == -1 {
// 		return nil, fmt.Errorf("low filter must be alpha:%v", filter)
// 	}
// 	if high = strings.Index(alpha, strings.ToLower(filterSplit[1])); high == -1 {
// 		return nil, fmt.Errorf("high filter must be alpha:%v", filter)
// 	}

// 	temp := min(low, high)
// 	high = max(low, high)
// 	low = temp
// 	filterMap := make(map[string]int)
// 	for i := low; i < high+1; i++ {
// 		filterMap[string(alpha[i])] = 0
// 	}
// 	return filterMap, nil
// }

// func okToPrint(filterMap map[string]int, label Visitor) bool {
// 	// If not filtering, don't print camps
// 	switch {
// 	case isEmpty(*filterType):
// 		{
// 			for key, data := range label {
// 				log.Println("key:", key, " data:", data)
// 				// do a special edit to make the reason for visit readable
// 				if isEmpty(*filterType) && key == "visitReason" && strings.Contains(data.(string), "makernexuscamp") {
// 					fmt.Printf("skipping label because this printer is not a camp printer")
// 					return false
// 				}
// 			}
// 			return true
// 		}
// 	case *filterType == "camp":
// 	}

// 	lastName := label["nameLast"].(string)
// 	firstChar := lastName[0:1]
// 	firstChar = strings.ToLower(firstChar)
// 	if _, exists := filterMap[firstChar]; !exists {
// 		fmt.Printf("skipping label.  lastName:%v filtermap:%v\n", lastName, filterMap)
// 		return false
// 	}
// 	return true
// }

// func isEmpty(s string) bool {
// 	return len(s) == 0
// }
