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
	Log        *debug.DebugClient
	connection *http.Client
	LabelDir   string `json:"labeldir"`
	Printers   map[string]Printer `json:"printers"`
	PrinterQueue []string           `json:"printqueue"`
	Current    int                `json:"current"`
	URL        string             `json:"url"`
	Reasons    []string           `json:"reasons"`
	FilterList map[string]string  `json:"filters"`
}

var clients map[string][]string
var filterList = map[string]string{
	"unedited":        "",
	"classOrworkshop": "Class/Workshop",
	"makernexusevent": "Special Event",
	"camp":            "Kids Camp",
	"volunteering":    "Volunteer",
	"forgotbadge":     "Forgot Badge",
	"guest":           "Visitor",
	"tour":            "Visitor",
	"other":           "Visitor",
}

func NewLabelClient(log *debug.DebugClient, dbURL string) *LabelClient {
	log.V(1).Printf("NewLabelClient started\n")
	l := new(LabelClient)
	var labelByte []byte
	l.Printers = make(map[string]Printer)
	l.Log = log
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Error getting user home directory:%v\n", err)
	}
	// change diretory to the label directory
	l.LabelDir = filepath.Join(home, "Documents", "Mylabels")
	if err := os.Chdir(l.LabelDir); err != nil {
		log.V(0).Printf("Label directory does not exist. path:%v\n", l.LabelDir)
		return nil
	}
	if _, err = os.Stat("labelConfig.json"); err != nil {
		log.V(0).Printf("Creating Configuration File")
		l.FilterList = filterList
		l.Reasons = make([]string, 0, 20)
		for key := range filterList {
			l.Reasons = append(l.Reasons, key)
		}
		l.URL = dbURL

	} else {
		byteJson, err := os.ReadFile("labelConfig.json")
		if err != nil {
			log.V(0).Fatalf("Error reading config file")
		}
		if err = json.Unmarshal(byteJson, l); err != nil {
			log.V(0).Fatalf("Error unmarshall labelConfig.json")
		}
	}
	// scan for the label printer in usb ports (brother or dymo)
	out, err := exec.Command("lsusb").CombinedOutput()
	if err != nil {
		log.Fatalf("exec.Command() failed error:%v\n", err)
	}
	log.V(1).Printf("lsusb:%v", string(out))
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
		case strings.Contains(line, "Brother"):
			brother_count += 1
		case strings.Contains(line, "LabelWriter"):
			dymo_count += 1
		}
	}
	log.V(0).Printf("Printers. Brother:%v Dymo:%v\n", brother_count, dymo_count)
	/*--------------------------------------------------------
	 * Get the output of "lpstat -p" and build a list of label
	 * printers.
	 *
	 * NOTE: lpstat shows print queues as ready even if the
	 *       printer is not attached
	 *---------------------------------------------------------*/
	out, err = exec.Command("lpstat", "-p").CombinedOutput()
	if err != nil {
		log.V(0).Fatalf("exec.Command() failed error:%v\n", err)
	}
	lines = strings.Split(string(out), "\n")
	if len(lines) == 0 {
		log.V(0).Fatalf("No Printers Found")
	}

loop:
	for _, line := range lines {
		model, _, status, err := l.parseLpstatP(line)
		if err != nil || status == "disabled" || model == "" {
			continue loop
		}
		log.V(1).Printf("model:%v status:%v\n", model, status)
		// set manufacturer.  Filter out the non-label printers
		// NOTE: there is a chance that a label printer was not
		// added as a cups printer.
		var p Printer
		switch {
		case strings.Contains(model, "QL") && brother_count > 0:
			p.PrinterManufacturer = "BROTHER"
			brother_count -= 1
		case strings.Contains(model, "LabelWriter") && dymo_count > 0:
			p.PrinterManufacturer = "DYMO"
			brother_count -= 1
		default:
			continue loop
		}
		p.PrinterModel = model
		p.CurrentJob = ""
		p.CurrentTime = time.Time{}
		p.PrinterStatus = status
		labelByte, err = os.ReadFile(p.PrinterManufacturer + ".glabels")
		if err != nil {
			log.Fatalf("The label template is missing.  Please create template the program glabels_qt. \nError:%v", err)
		}
		p.LabelTemplate = string(labelByte)
		p.JobQueue = make([]Job, 0)
		log.V(1).Printf("add printer [%v:%v]\n", p.PrinterManufacturer, p.PrinterModel)
		l.Printers[model] = p
		l.PrinterQueue = append(l.PrinterQueue, model)
	} // for each line in lpstat -p
	/*---------------------------------------------------------
	 * write the label config to disk
	 *-------------------------------------------------------*/
	var configBuf []byte
	if configBuf, err = json.Marshal(l); err != nil {
		log.V(0).Fatalf("Error marshal labelConfig.json err:%v", err)
	}
	if err := os.WriteFile("labelConfig.json", configBuf, 0777); err != nil {
		log.V(0).Fatalf("Error writing labelConfig.json err:%v", err)
	}
	/*----------------------------------------------------------
	 * Stop program to make them edit the label config filters
	 * if this is a brand new config file.  We added a list of
	 * all reason codes to make it easier
	 *---------------------------------------------------------*/
	if l.Reasons[0] == "unedited" {
		log.V(0).Printf(`Please edit "~\Documents\Mylabels\labelConfig.json" and modify the list of reasons that will be received at this printer`)
		log.V(0).Fatalf("Don't forget to remove unedited from the reason")
	}
	// Create the http client with no security this is to access the OVL database
	// website
	l.connection = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	log.V(1).Printf("Return NewLabelClient\n")
	return l
}

func (l *LabelClient) ReadOVL(url string) ([]Visitor, error) {
	// Do the http.get
	log := l.Log
	labels := new(VisitorList)
	html, err := l.connection.Get(url)
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
func (l *LabelClient) ExportToGlabels(info Visitor) error {

	var model string
	var p Printer
	var exist bool
	if l.Current >= len(l.PrinterQueue) {
		l.Current = 0
	}

	if p, exist = l.Printers[model]; !exist {
		return fmt.Errorf("missing printer model:%v", model)
	}
	// Get the date right now and update the label
	temp := p.LabelTemplate
	t := time.Now()
	nowDate := fmt.Sprintf("%v %v, %v", toMonth[t.Month()], t.Day(), t.Year())
	temp = strings.Replace(temp, "${Date}", nowDate, -1)
	// Find the first and last name
	var first, last, reason string

	for key, data := range info {
		log.V(1).Printf("key:%v\n", key)
		switch key {
		case "nameFirst":
			log.V(1).Printf("found first:%v\n", data.(string))
			temp = strings.Replace(temp, "${nameFirst}", data.(string), -1)
			first = data.(string)
		case "nameLast":
			log.V(1).Printf("found last:%v\n", data.(string))
			temp = strings.Replace(temp, "${nameLast}", data.(string), -1)
			last = data.(string)
		case "visitReason":
			var exist bool
			if reason, exist = filterList[data.(string)]; !exist {
				reason = "Visitor"
			}
			if reason == "Forgot Badge" {
				key := strings.ToLower(last + first)
				if c, exist := clients[key]; exist {
					reason = "Member"
					s := strings.ToLower(c[9])
					if strings.Contains(s, "staff") {
						reason = "Staff"
					}
				}
			}
			temp = strings.Replace(temp, "Visitor", reason, -1)
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
	return nil
}
func (l *LabelClient) ProcessLabelQueue() (err error) {
	log := l.Log
	// cd to the Mylabel directory so we can write files
	if err := os.Chdir(l.LabelDir); err != nil {
		log.Fatal("Label directory does not exist.")
	}
	/*----------------------------------------------------------------
	 * Print all in the queue
	 * -------------------------------------------------------------*/
	var out []byte
	var lines []string
	for model, p := range l.Printers {
		log.V(2).Printf("processLabels there are %v printers. key/model:%v length:%v\n", len(l.Printers), model, len(p.JobQueue))
		for _, j := range p.JobQueue {
			// delete and write the temp.glables file
			os.Remove("temp.glabels")
			if err := os.WriteFile("temp.glabels", []byte(j.Label), 0666); err != nil {
				return fmt.Errorf("write to temp.glabels error:%v\n", err)
			}
			if out, err = exec.Command("glabels-batch-qt", "--printer="+p.PrinterModel, "temp.glabels").CombinedOutput(); err != nil {
				return fmt.Errorf("glabels-batch-qt --printer=%v temp.glabels\n  error:%v\n  output:%v\n", p.PrinterModel, err, string(out))
			}
			j.Start = time.Now()
			/*-----------------------------------------------------
			 * Get the job number of the last print.  Should be the
			 * last entry in the lpstat output
			 *----------------------------------------------------*/
			if out, err = exec.Command("lpstat", "-W", "not-completed").CombinedOutput(); err != nil || len(out) == 0 {
				log.V(0).Printf("lpstat -W all\n  error:%v\n  output:%v\n", err, string(out))
			}
			lines := strings.Split(string(out), "\n")
			// Debug print
			for i, line := range lines {
				log.V(1).Printf("%v lpstat:%v\n", i, line)
			}
			line := lines[len(lines)-2]
			tokens := strings.Split(line, " ")
			j.JobNumber = tokens[0]
			log.V(0).Printf("PRINTING... [%v] name: %v %v reason:%v\n", j.JobNumber, j.First, j.Last, j.Reason)

		}
		p.JobQueue = make([]Job, 0) // clear the job queue
		l.Printers[model] = p       // store this printer in printer slice
	}
	/*-----------------------------------------------------------------
	 * Update printer status
	 *---------------------------------------------------------------*/
	if out, err = exec.Command("lpstat", "-p").CombinedOutput(); err != nil || len(out) == 0 {
		log.V(0).Printf("lpstat -W all\n  error:%v\n  output:%v\n", err, string(out))
	}
	lines = strings.Split(string(out), "\n")
	for _, line := range lines {
		model, job, status, err := l.parseLpstatP(line)
		switch {
		case err != nil,
			!strings.Contains(model, "QL") && !strings.Contains(model, "LabelWriter"):
			continue
		}

		// Check each printer to see if there is a new job printing or idle
		p := l.Printers[model]
		p.PrinterStatus = status
		switch {
		case strings.Contains(status, "printing") && p.CurrentJob != job:
			p.CurrentJob = job
			p.CurrentTime = time.Now()
		case strings.Contains(status, "idle"):
			p.CurrentJob = ""
			p.CurrentTime = time.Time{} // should be 0 time
		}
		l.Printers[model] = p
	} // for each line in "lpstat -p"

	return nil
}
func (l *LabelClient) WaitTillIdle(seconds int) (isidle bool) {
	log := l.Log
	/*----------------------------------------------------------------
	 * Print all in the queue
	 * -------------------------------------------------------------*/
	var out []byte
	var lines []string
	isidle = false
	for i := 0; i < seconds && !isidle; i++ {
		isidle = true
		if out, err := exec.Command("lpstat", "-p").CombinedOutput(); err != nil || len(out) == 0 {
			log.V(0).Printf("lpstat -W all\n  error:%v\n  output:%v\n", err, string(out))
		}
		lines = strings.Split(string(out), "\n")
		for _, line := range lines {
			model, _, status, err := l.parseLpstatP(line)
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

func (c *LabelClient) parseLpstatP(line string) (model, job, status string, err error) {
	sentences := strings.Split(line, ".")
	if len(sentences) < 2 {
		return "", "", "", fmt.Errorf("line does not have 2 sentences:%v", line)
	}
	// split first sentence into words, extract the printer model
	words := strings.Split(sentences[0], " ")
	if len(words) < 4 {
		return "", "", "", fmt.Errorf("first sentence is too short len:%v", len(words))
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
		job = words[len(words)-1] // job is the last word in the sentence
		status = "printing"
	} // switch

	return

}

// Monitor print queues for all the printers and wait till they are
// empty
func (c *LabelClient) IsStuck(p Printer) bool {
	if !p.CurrentTime.IsZero() && time.Since(p.CurrentTime) > 60*time.Second {
		return true
	}
	return false
}

// Print a test page to each printer
func (l *LabelClient) ExportTestToGlabels() error {
	for _, p := range l.Printers {
		var temp string = p.LabelTemplate
		// Get the date right now and update the label
		t := time.Now()
		nowDate := fmt.Sprintf("%v %v, %v", toMonth[t.Month()], t.Day(), t.Year())
		temp = strings.Replace(temp, "${Date}", nowDate, -1)
		temp = strings.Replace(temp, "${nameFirst}", "WELCOME TO MAKERNEXUS", -1)
		temp = strings.Replace(temp, "${nameLast}", "", -1)
		temp = strings.Replace(temp, "Visitor", "Test Label", -1)
		// cd to the Mylabel directory so we can write files
		if err := os.Chdir(l.LabelDir); err != nil {
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

// Create directories
func (l *LabelClient) dirSetup() error {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	configPath := filepath.Join(home, "Documents", "Mylabels")
	if err := os.Chdir(configPath); err != nil {
		return fmt.Errorf("error changing to home directory")
	}
	/*----------------------------------------------------------------
	 * if directory does not exist then create it
	 *----------------------------------------------------------------*/
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.Mkdir(configPath, 0777); err != nil {
			return fmt.Errorf("error creating directory %v", configPath)
		}
	}
	return nil
}
