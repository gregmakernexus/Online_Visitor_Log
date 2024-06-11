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

type Job struct {
   JobNumber string
   Start     time.Time
   Label     string	
   First     string
   Last      string
   Status    string
   Alerts    string
   Queued    string
   Reason    string
   
}
type Printer struct {
	LabelTemplate       string
	PrinterModel        string
	PrinterManufacturer string
	PrinterStatus       string
	JobQueue            []Job
	CurrentJob          string
	CurrentTime         time.Time
}
type LabelClient struct {
	LabelDir   string
	connection *http.Client
	Printers   map[string]Printer
	PrinterQueue []string
	Current    int
	Log        *debug.DebugClient
}

var clients map[string][]string

func NewLabelClient(log *debug.DebugClient) *LabelClient {
	log.V(2).Printf("NewLabelClient started\n")
	c := new(LabelClient)
	c.Log = log
	var labelByte []byte
	c.Printers = make(map[string]Printer)
	
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Error getting user home directory:%v\n", err)
	}
	// Read the label file into a string
	c.LabelDir = filepath.Join(home, "Mylabels")
	if err := os.Chdir(c.LabelDir); err != nil {
		log.Fatalf("Label directory does not exist. path:%v\n", c.LabelDir)
	}
	
	// list usb devices
	out, err := exec.Command("lsusb").CombinedOutput()
	if err != nil {
		log.Fatalf("exec.Command() failed error:%v\n", err)
	}
	log.V(1).Printf("lsusb:%v",string(out))
	lines := strings.Split(string(out), "\n")
	if len(lines) == 0 {
		log.Fatalf("No Printers Found")
	}
	// count the number of brother or dymo printers that are plugged 
	// into usb
	dymo_count := 0
	brother_count := 0
	for _, line := range lines {
	  switch {
		  case strings.Contains(line,"Brother"):
		    brother_count += 1
		  case strings.Contains(line,"LabelWriter"):
		    dymo_count += 1
      }
	}
	log.V(0).Printf("Printers. Brother:%v Dymo:%v\n",brother_count,dymo_count)
	// Find out if cups driver is installed for glabels.  lpstat will
	// show the printer as ready even if it is not plugged in.
	out, err = exec.Command("lpstat", "-p").CombinedOutput()
	if err != nil {
		log.Fatalf("exec.Command() failed error:%v\n", err)
	}
	lines = strings.Split(string(out), "\n")
	if len(lines) == 0 {
		log.Fatalf("No Printers Found")
	}
	
	loop:
	for _, line := range lines {
		model, _, status, err := c.parseLpstatP(line)	
		if err != nil || status == "disabled" || model == "" {
			continue loop
		}
		
		log.V(1).Printf("model:%v status:%v\n",model, status)
		// set manufacturer
		p := c.Printers[model]
		switch {
		case strings.Contains(model, "QL"):
		    if brother_count <= 0 {
				continue loop
			}
		    p.PrinterManufacturer = "BROTHER"
		    brother_count -= 1
		case strings.Contains(model, "LabelWriter"):
		    if dymo_count <= 0 {
				continue loop
			}
			p.PrinterManufacturer = "DYMO"
			dymo_count -= 1
		default:
		    continue loop
		}
		
		p.PrinterModel = model
		p.CurrentJob   = ""
		p.CurrentTime  = time.Time{}
		p.PrinterStatus = status
		labelByte, err = os.ReadFile(p.PrinterManufacturer + ".glabels")
		if err != nil {
			log.Fatalf("The label template is missing.  Please create template the program glabels_qt. \nError:%v", err)
		}
		p.LabelTemplate = string(labelByte)
		p.JobQueue = make([]Job,0)
		
		log.V(1).Printf("add printer [%v:%v]\n", p.PrinterManufacturer,p.PrinterModel)
		c.Printers[model] = p
		c.PrinterQueue = append(c.PrinterQueue,model) 
	} // for
	c.Current  = 0
	
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
	
	log.V(2).Printf("Return NewLabelClient\n")
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
func (c *LabelClient) AddToLabelQueue(info Visitor) error {
	log := c.Log
	if len(info) == 0 {
		return nil
	}
	if c.Current >= len(c.PrinterQueue) {
	  log.V(1).Printf("c.Current:%v len(c.PrinterQueue):%v\n",c.Current,len(c.PrinterQueue))
	  c.Current = 0
	}
	model := c.PrinterQueue[c.Current]
	p := c.Printers[model]
	var temp string = c.Printers[model].LabelTemplate
	// Get the date right now and update the label
	t := time.Now()
	nowDate := fmt.Sprintf("%v %v, %v", toMonth[t.Month()], t.Day(), t.Year())
	temp = strings.Replace(temp, "${Date}", nowDate, -1)
	// Find the first and last name
	var first, last, reason string
	for key, data := range info {
		log.V(1).Printf("key:%v\n",key)
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
				reason = "Event"
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
	log.V(1).Printf("replace Visitor with:%v.\n",reason)
	temp = strings.Replace(temp, "Visitor", reason, -1)
	
	/*-----------------------------------------------------------------
	 * Store the label in the queue
	 *----------------------------------------------------------------*/
    var j Job
    j.Label = temp
    j.First = first
    j.Last = last
    j.Start = time.Now()
    j.Reason = reason
    p.JobQueue = append(p.JobQueue, j)
    log.V(2).Printf("Print to %v Start *******************\n",p.PrinterModel)
    for i,d := range p.JobQueue {
		log.V(2).Printf("%v name:%v %v\n",i, d.First,d.Last)
    }
    c.Printers[p.PrinterModel] = p
	/*---------------------------------------------------------------
	 * If there is multiple printers, prepare for next print in a round
	 * robin fashion
	 *---------------------------------------------------------------*/ 
	
	c.Current++
	if c.Current > len(c.PrinterQueue)-1 {
		c.Current = 0
	}
	log.V(2).Printf("Next Printer:%v\n",c.PrinterQueue[c.Current])
	
	return nil
}

func (c *LabelClient) ProcessLabelQueue() (err error) {
	log := c.Log
	// cd to the Mylabel directory so we can write files
	if err := os.Chdir(c.LabelDir); err != nil {
		return fmt.Errorf("%v directory does not exist",c.LabelDir)
	}
	/*----------------------------------------------------------------
	 * Print all in the queue
	 * -------------------------------------------------------------*/
	var out []byte
	var lines []string
	for model,p:= range c.Printers {
	    log.V(2).Printf("processLabels there are %v printers. key/model:%v length:%v\n",len(c.Printers), model, len(p.JobQueue))
	    for _,j := range p.JobQueue {
			// delete and write the temp.glables file
			os.Remove("temp.glabels")
			if err := os.WriteFile("temp.glabels", []byte(j.Label), 0666); err != nil {
				return fmt.Errorf("write to temp.glabels error:%v\n", err)
			}
			if out, err = exec.Command("glabels-batch-qt", "--printer="+p.PrinterModel, "temp.glabels").CombinedOutput(); err != nil {
				return fmt.Errorf("glabels-batch-qt --printer=%v temp.glabels\n  error:%v\n  output:%v\n", p.PrinterModel, err, string(out) )
			}
			j.Start = time.Now()
			/*-----------------------------------------------------
			 * Get the job number of the last print.  Should be the
			 * last entry in the lpstat output
			 *----------------------------------------------------*/
			if out, err = exec.Command("lpstat", "-W", "not-completed").CombinedOutput(); err != nil || len(out) == 0 {
				log.V(0).Printf("lpstat -W all\n  error:%v\n  output:%v\n", err, string(out))
			}
			lines := strings.Split(string(out),"\n")
			// Debug print
			for i,line := range lines {
				log.V(1).Printf("%v lpstat:%v\n",i,line)
			}
			line := lines[len(lines)-2]
			tokens := strings.Split(line, " ")
			j.JobNumber = tokens[0]
			log.V(0).Printf("PRINTING... [%v] name: %v %v reason:%v\n",j.JobNumber,j.First, j.Last,j.Reason)
			
		}	
        p.JobQueue = make([]Job,0)  // clear the job queue
        c.Printers[model] = p      // store this printer in printer slice
	}
    /*-----------------------------------------------------------------
     * Update printer status
     *---------------------------------------------------------------*/
	if out, err = exec.Command("lpstat", "-p").CombinedOutput(); err != nil || len(out) == 0 {
		log.V(0).Printf("lpstat -W all\n  error:%v\n  output:%v\n", err, string(out))
	}
	lines = strings.Split(string(out),"\n")
	for _,line := range lines {
		model,job,status,err := c.parseLpstatP(line)	
		switch {
		  case  err != nil,
		    !strings.Contains(model, "QL") && !strings.Contains(model, "LabelWriter"):
		  continue
		}
		
		// Check each printer to see if there is a new job printing or idle
		p := c.Printers[model]
		p.PrinterStatus = status
		switch {
		case strings.Contains(status,"printing") && p.CurrentJob != job:
			p.CurrentJob = job
			p.CurrentTime = time.Now()
		case strings.Contains(status,"idle"):
			p.CurrentJob = ""
			p.CurrentTime = time.Time{}  // should be 0 time
		}
	    c.Printers[model] = p
	} // for each line in "lpstat -p"
	
	return nil	
}
func (c *LabelClient) WaitTillIdle(seconds int) (isidle bool) {
	log := c.Log
	/*----------------------------------------------------------------
	 * Print all in the queue
	 * -------------------------------------------------------------*/
	var out []byte
	var lines []string
	isidle = false
	for i:=0;i<seconds && !isidle;i++ {
		isidle = true
		if out, err := exec.Command("lpstat", "-p").CombinedOutput(); err != nil || len(out) == 0 {
			log.V(0).Printf("lpstat -W all\n  error:%v\n  output:%v\n", err, string(out))
		}
		lines = strings.Split(string(out),"\n")
		for _,line := range lines {
			model,_,status,err := c.parseLpstatP(line)	
			if err != nil {
				continue
			}
			// Check each printer to see if there is a job printing
			if (strings.Contains(model, "QL") || strings.Contains(model, "LabelWriter")) &&
				status == "printing" {
				isidle = false
				break
			}
		}
		
		time.Sleep(1)
	}
	return
}

func (c *LabelClient) parseLpstatP(line string) (model, job, status string,err error) {
	sentences := strings.Split(line,".")
	if len(sentences) < 2 {
		return "","", "", fmt.Errorf("line does not have 2 sentences:%v",line)
	}
	// split first sentence into words, extract the printer model
	words := strings.Split(sentences[0], " ")
	if len(words) < 4 {
		return "","","", fmt.Errorf("first sentence is too short len:%v", len(words))
	}
	model = words[1] 
	switch {
		// Printer is disabled or idle
		case words[len(words)-1] == "disabled",
		    words[len(words)-1] == "idle":
            job = ""
		    status = words[len(words)-1] 
		// Printer is printing
		case strings.Contains(line, "printing"):
			job   = words[len(words)-1]    // job is the last word in the sentence      
		    status = "printing"
	} // switch
	
	return 
	
	
	
}
// Monitor print queues for all the printers and wait till they are
// empty
func (c *LabelClient) IsStuck(p Printer) bool {
	if !p.CurrentTime.IsZero() && time.Since(p.CurrentTime) > 60 * time.Second {
		 return true
	}
	return false
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
