// label takes the data form OVL database and exports it to .glabels file
package label

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	// "net/http/httputil"

	"example.com/debug"
)

// So we can have one set of code we have a set of filters to allow the following scenarios
//  1. --filter=printers.  Multiple printers for a large event.  We segment the alphabet and only print the specified range
//  2. --filter=camp.      Summer camp.
//  3. no filter specified This is the printer at the normal mod station.  If we are using #1 above.  This printer
//     must be power down.
var toMonth = []string{"", "Jan.", "Feb.", "Mar.", "Apr.", "May", "Jun.", "Jul.", "Aug.", "Sep.", "Oct.", "Nov.", "Dec."}

// var log *debug.DebugClient

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
	Log          *debug.DebugClient
	connection   *http.Client
	Clients      map[string][]string
	BrotherCount int
	DymoCount    int
	LabelDir     string             `json:"labeldir"`
	Printers     map[string]Printer `json:"printers"`
	PrinterQueue []string           `json:"printqueue"`
	URL          string             `json:"url"`
	URLWithParms string             `json:"urlwithparms"`
	Reasons      []string           `json:"reasons"`
	FilterList   map[string]string  `json:"filters"`
}

var filterList = map[string]string{
	"classOrworkshop": "Class",
	"makernexusevent": "Event",
	"camp":            "Kids Camp",
	"volunteering":    "Volunteer",
	"forgotbadge":     "Forgot Badge",
	"guest":           "Guest",
	"tour":            "TOUR",
	"other":           "Visitor",
	"meetup":          "Meet-Up",
}

func NewLabelClient(log *debug.DebugClient, dbURL string) *LabelClient {
	l := new(LabelClient)
	l.Log = log
	log.V(1).Printf("NewLabelClient started\n")
	l.Printers = make(map[string]Printer)

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Error getting user home directory:%v\n", err)
	}
	/*----------------------------------------------------------------
	 * Read the config file.  If not there, build default.
	 *---------------------------------------------------------------*/
	l.LabelDir = filepath.Join(home, "Mylabels")
	config := filepath.Join(home, ".makernexus")
	if err := os.Chdir(config); err != nil {
		log.V(0).Printf(".makernexus directory does not exist. path:%v\n", config)
		l.dirSetup(".makernexus")
	}
	if _, err = os.Stat("labelConfig.json"); err != nil {
		log.V(0).Printf("Creating Configuration File\n")
		l.FilterList = filterList // default is all catagories
		l.Reasons = make([]string, 0, 20)

	} else {
		byteJson, err := os.ReadFile("labelConfig.json")
		if err != nil {
			log.V(0).Fatalf("Error reading config file")
		}
		if err = json.Unmarshal(byteJson, l); err != nil {
			log.V(0).Fatalf("Error unmarshall labelConfig.json")
		}
	}
	/*----------------------------------------------------------------
	 * Build the URL so the php server will read parms
	 *---------------------------------------------------------------*/
	l.URL = dbURL
	l.URLWithParms = l.URL + "?vrss="
	for key := range l.FilterList {
		l.Reasons = append(l.Reasons, key)
		l.URLWithParms = l.URLWithParms + key + "|"
	}
	l.URLWithParms = l.URLWithParms[:len(l.URLWithParms)-1]
	log.V(1).Printf("DB Url:%v\n", l.URLWithParms)
	/*-----------------------------------------------------------------
	 * Count the number of label printers attached to USB (lsusb)
	 *----------------------------------------------------------------*/
	l.CountUSBPrinters()
	/*------------------------------------------------------------------
	 * Get CUPS printers
	 *---------------------------------------------------------*/
	l.Printers = make(map[string]Printer) // reinitialie in case templates change
	l.PrinterQueue = make([]string, 0)
	/*---------------------------------------------------------
	 * write the label config to disk
	 *-------------------------------------------------------*/
	var configBuf []byte
	l.Clients = make(map[string][]string)
	if err := os.Chdir(config); err != nil {
		log.V(0).Printf(".makernexus directory does not exist. path:%v\n", config)
		l.dirSetup(".makernexus")
	}
	if configBuf, err = json.Marshal(l); err != nil {
		log.V(0).Fatalf("Error marshal labelConfig.json err:%v", err)
	}
	if err := os.WriteFile("labelConfig.json", configBuf, 0777); err != nil {
		log.V(0).Fatalf("Error writing labelConfig.json err:%v", err)
	}
	l.updateCUPSPrinter()

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
	log.V(1).Printf("http get parms:%v\n", l.URLWithParms)
	html, err := l.connection.Get(l.URLWithParms)
	if err != nil {
		log.Fatal(err)
	}

	// extract the body from the http packet
	results, err := io.ReadAll(html.Body)
	if err != nil {
		log.Fatal(err)
	}
	html.Body.Close()
	log.V(1).Printf("ReadOVL results:%v\n", string(results))
	// unmarshall the json object
	if err := json.Unmarshal(results, labels); err != nil {
		fmt.Printf("unmarshall error:%v\n", err)
		return nil, err
	}
	for i, v := range labels.Data.Visitors {
		log.V(1).Printf("%v %v %v reason:%v\n", i, v["nameFirst"], v["nameLast"], v["visitReason"])
	}
	return labels.Data.Visitors, nil

}

func (l *LabelClient) AddToLabelQueue(info Visitor) error {
	log := l.Log
	if len(info) == 0 {
		return nil
	}
	var first, last, reason, status string
	var p Printer
	var exist bool
	var reasons map[string]string = make(map[string]string)
	if len(l.PrinterQueue) == 0 {
		l.updateCUPSPrinter()
	}
	var model string = l.PrinterQueue[0] // get printer from printer queue
	if p, exist = l.Printers[model]; !exist {
		return fmt.Errorf("missing printer model:%v", model)
	}
	// Get the date right now and update the label
	temp := p.LabelTemplate // make a copy
	t := time.Now()
	nowDate := fmt.Sprintf("%v %v, %v", toMonth[t.Month()], t.Day(), t.Year())
	temp = strings.Replace(temp, "${Date}", nowDate, -1)
	/*-----------------------------------------------------------------
	 * Since the order of the keys is random, parse all the keys
	 *---------------------------------------------------------------*/
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
			for _, r := range strings.Split(data.(string), "|") {
				r = strings.Trim(r, " ")
				log.V(1).Printf("Adding Reason key:%v data:%v\n", r, filterList[r])
				reasons[r] = filterList[r]
			}
			log.V(1).Printf("Reasons:%v\n", reasons)
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
	/*----------------------------------------------------------------
	 * Determine membership status from the rfid table
	 *--------------------------------------------------------------*/
	status = "Visitor"
	key := strings.ToLower(last + first)
	if c, exist := l.Clients[key]; exist {
		status = "Member"
		if len(c) > 7 {
			s := strings.ToLower(c[7])
			if strings.Contains(s, "staff") {
				status = "Staff"
			}
		}
	}
	/*------------------------------------------------------------------
	 *  Replace "Visitor" with other visitor reasons
	 *-----------------------------------------------------------------*/
	if status == "Visitor" {
		if s, exist := reasons["volunteering"]; exist {
			status = s
		}
		if s, exist := reasons["guest"]; exist {
			status = s
		}
	}
	/*------------------------------------------------------------------
	 *  Handle special reasons for the visit
	 *-----------------------------------------------------------------*/
	reason = ""
	if r, exist := reasons["classOrworkshop"]; exist {

		reason = appendString(reason, r)
		log.V(2).Printf("classorworkship exists reason:%v\n", reason)
	}
	if r, exist := reasons["makernexusevent"]; exist {

		reason = appendString(reason, r)
		log.V(2).Printf("event exists reason:%v\n", reason)
	}
	if r, exist := reasons["camp"]; exist {
		status = "Student"
		reason = appendString(reason, r)
		log.V(2).Printf("camp exists reason:%v\n", reason)
	}
	if r, exist := reasons["other"]; exist {
		reason = appendString(reason, r)
		log.V(2).Printf("other exists reason:%v\n", reason)
	}
	if r, exist := reasons["tour"]; exist {
		reason = appendString(reason, r)
		log.V(2).Printf("tour exists reason:%v\n", reason)
	}
	if r, exist := reasons["meetup"]; exist {
		reason = appendString(reason, r)
		log.V(2).Printf("meetup exists reason:%v\n", reason)
	}
	log.V(1).Printf("Reason:%v\n", reason)
	temp = strings.Replace(temp, "${reason}", reason, -1)
	log.V(1).Printf("Member Status:%v\n", status)
	temp = strings.Replace(temp, "Visitor", status, -1)
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
	l.Printers[p.PrinterModel] = p
	/*-----------------------------------------------------------------
	 * pop the queue and add it to the end
	 *----------------------------------------------------------------*/
	l.PrinterQueue = l.PrinterQueue[1:]
	l.PrinterQueue = append(l.PrinterQueue, p.PrinterModel)
	/*-----------------------------------------------------------------
	 * debug messages
	 *-----------------------------------------------------------------*/
	log.V(2).Printf("Print to %v Start *******************\n", p.PrinterModel)
	for i, d := range p.JobQueue {
		log.V(2).Printf("%v name:%v %v\n", i, d.First, d.Last)
	}

	return nil
}
func appendString(s string, a string) string {
	if s != "" {
		s = s + ","
	}
	s = s + a
	return s
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
	for model, p := range l.Printers {
		log.V(2).Printf("processLabels there are %v printers. key/model:%v jobqueue length:%v\n", len(l.Printers), model, len(p.JobQueue))
		if len(l.Printers) > 1 {
			log.V(0).Printf("more than one printer:%v\n", l.Printers)
		}
		for _, j := range p.JobQueue {
			// delete and write the temp.glables file
			os.Remove("temp.glabels")
			if err := os.WriteFile("temp.glabels", []byte(j.Label), 0666); err != nil {
				return fmt.Errorf("write to temp.glabels error:%v", err)
			}
			// Then print it using the glabels-batch.qt command
			if out, err = exec.Command("glabels-batch-qt", "--printer="+p.PrinterModel, "temp.glabels").CombinedOutput(); err != nil {
				return fmt.Errorf("glabels-batch-qt --printer=%v temp.glabels  error:%v  output:%v", p.PrinterModel, err, string(out))
			}
			// j.Start = time.Now()
			j.JobNumber = l.getJobNumber()
			log.V(0).Printf("PRINTING... [%v] name: %v %v reason:%v\n", j.JobNumber, j.First, j.Last, j.Reason)

		}
		p.JobQueue = make([]Job, 0) // clear the job queue
		l.Printers[model] = p       // store this printer in printer slice
	}
	/*-----------------------------------------------------------------
	 * Update printer status
	 *---------------------------------------------------------------*/
	l.updateCUPSPrinter()
	return nil
}

func (l *LabelClient) parseLpstatP(line string) (model, job, status string, err error) {
	// log := l.Log  // not used
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

// Print a test page to each printer
func (l *LabelClient) ExportTestToGlabels() error {
	log := l.Log
	// cd to Mylabels
	home, _ := os.UserHomeDir()
	if err := os.Chdir(filepath.Join(home, "Mylabels")); err != nil {
		log.V(0).Printf("Label directory does not exist. path:%v\n", l.LabelDir)
		l.dirSetup("Mylabels")
	}
	// read the printer diagostic label
	labelByte, err := os.ReadFile("printer.glabels")
	if err != nil {
		log.Fatalf("The label template is missing.  Please create template the program glabels_qt. \nError:%v", err)
	}
	template := string(labelByte)
	if len(l.Printers) == 0 {
		l.updateCUPSPrinter()
	}
	for _, p := range l.Printers {
		var temp string = template
		// Get the date right now and update the label
		t := time.Now()
		nowDate := fmt.Sprintf("%v %v, %v", toMonth[t.Month()], t.Day(), t.Year())
		temp = strings.Replace(temp, "${Date}", nowDate, -1)
		// print the filterlist (up to 10)
		i := 1
		for key := range l.FilterList {
			s := fmt.Sprintf("${filter%v}", i)
			temp = strings.Replace(temp, s, key, -1)
			i++
		}
		// cd to the Mylabel directory so we can write files
		if err := os.Chdir(l.LabelDir); err != nil {
			log.Fatal("Label directory does not exist.")
		}
		// delete and write the temp.glables file
		os.Remove("temp.glabels")
		if err := os.WriteFile("temp.glabels", []byte(temp), 0666); err != nil {
			log.Fatalf("Error writing label file error:%v\n", err)
		}
		if out, err := exec.Command("glabels-batch-qt", "--printer="+p.PrinterModel, "temp.glabels").CombinedOutput(); err != nil {
			return fmt.Errorf("glabels-batch-qt --printer=%v temp.glabels  error:%v  output:%v", p.PrinterModel, err, string(out))
		}
	}
	return nil
}

// Create directories
func (l *LabelClient) dirSetup(name string) error {
	log := l.Log
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	configPath := filepath.Join(home, name)
	/*----------------------------------------------------------------
	 * if directory does not exist then create it
	 *----------------------------------------------------------------*/
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.Mkdir(configPath, 0777); err != nil {
			return fmt.Errorf("error creating directory %v", configPath)
		}
	}
	if err := os.Chdir(configPath); err != nil {
		return fmt.Errorf("error changing to home directory")
	}
	return nil
}

func (l *LabelClient) FilterEditor(input *os.File, output *os.File, echo bool) (err error) {
	log := l.Log
	rdr := bufio.NewReader(input)
	w := tabwriter.NewWriter(output, 0, 0, 1, ' ', 0)

	/*----------------------------------------------------------------
	 * Courtesy print of the filter list
	 * --------------------------------------------------------------*/
	fmt.Fprintf(w, "filter\tdisplayed reason\n")
	fmt.Fprintf(w, "------\t----------------\n")
	for key, value := range l.FilterList {
		fmt.Fprintf(w, "%v\t%v\n", key, value)
	}
	w.Flush()
	for {
		fmt.Fprintf(output, "Enter Command: (cr when done):")
		line, _ := rdr.ReadString('\n')
		if echo {
			fmt.Fprintf(output, "%v", line)
		}
		line = strings.Replace(line, "\n", "", -1)
		if len(line) == 0 {
			continue
		}
		line = strings.ToLower(line)
		tokens := strings.Split(line, " ")
		switch {
		case tokens[0] == "add", tokens[0] == "a":
			if len(tokens) < 2 {
				fmt.Fprintf(output, "missing parameter")
				continue
			}
			if _, exist := l.FilterList[tokens[1]]; exist {
				fmt.Fprintf(output, "Filter %v already exists\n", tokens[1])
				continue
			}
			if reason, exist := filterList[tokens[1]]; exist {
				l.FilterList[tokens[1]] = reason
				continue
			}
			fmt.Fprintf(output, "Adding a unknow filter.  Enter reason text:")
			line, _ := rdr.ReadString('\n')
			if echo {
				fmt.Fprintf(output, "%v", line)
			}
			line = strings.Replace(line, "\n", "", -1)
			if len(line) > 0 {
				l.FilterList[tokens[1]] = line
			}

		case tokens[0] == "delete", tokens[0] == "d":
			if len(tokens) < 2 {
				fmt.Fprintf(output, "missing parameter")
				continue
			}
			if _, exist := l.FilterList[tokens[1]]; !exist {
				fmt.Fprintf(output, "filter does not exist\n")
				continue
			}
			delete(l.FilterList, tokens[1])
		case tokens[0] == "modify", tokens[0] == "m":
			if len(tokens) < 2 {
				fmt.Fprintf(output, "missing parameter")
				continue
			}
			if _, exist := l.FilterList[tokens[1]]; !exist {
				fmt.Fprintf(output, "filter does not exist\n")
				continue
			}
			fmt.Fprintf(output, "Modifying %v. Enter new reason:", tokens[1])
			line, _ := rdr.ReadString('\n')
			if echo {
				fmt.Fprintf(output, "%v", line)
			}
			line = strings.Replace(line, "\n", "", -1)
			if len(line) > 0 {
				l.FilterList[tokens[1]] = line
			}
		case tokens[0] == "list", tokens[0] == "l":
			fmt.Fprintf(w, "filter\tdisplayed reason\n")
			fmt.Fprintf(w, "------\t----------------\n")
			for key, value := range l.FilterList {
				fmt.Fprintf(w, "%v\t%v\n", key, value)
			}
			w.Flush()
		case tokens[0] == "clear", tokens[0] == "c":
			l.FilterList = make(map[string]string)
		case tokens[0] == "reset", tokens[0] == "r":
			l.FilterList = filterList
		case tokens[0] == "exit", tokens[0] == "e":
			// Update db read parameters
			l.URLWithParms = l.URL + "?vrss="
			l.Reasons = make([]string, 0)
			for key := range l.FilterList {
				l.Reasons = append(l.Reasons, key)
				l.URLWithParms = l.URLWithParms + key + "|"
			}
			l.URLWithParms = l.URLWithParms[:len(l.URLWithParms)-1]
			log.V(1).Printf("DB Url:%v\n", l.URLWithParms)
			// Don't store printer or client information
			l.Clients = make(map[string][]string)
			l.Printers = make(map[string]Printer)
			l.PrinterQueue = make([]string, 0)
			// Write config file to disk
			var configBuf []byte
			if configBuf, err = json.Marshal(l); err != nil {
				log.V(0).Fatalf("Error marshal labelConfig.json err:%v", err)
			}
			if err := os.WriteFile("labelConfig.json", configBuf, 0777); err != nil {
				log.V(0).Fatalf("Error writing labelConfig.json err:%v", err)
			}
			fmt.Fprintf(output, "\n")
			return nil
		case tokens[0] == "help", tokens[0] == "h":
			fmt.Fprintf(w, "+--------------------\t+-------------------------------------------------------------\t+\n")
			fmt.Fprintf(w, "| command\t| description\t|\n")
			fmt.Fprintf(w, "+--------------------\t+-------------------------------------------------------------\t+\n")
			fmt.Fprintf(w, "| add filtername\t| print labels with this reason on this printer\t|\n")
			fmt.Fprintf(w, "| delete filtername\t| do not print specified labels on this printer\t|\n")
			fmt.Fprintf(w, "| clear\t| remove all filters, which will not print anything\t|\n")
			fmt.Fprintf(w, "| list\t| list all the filters and reasons\t|\n")
			fmt.Fprintf(w, "| reset\t| add all filters, which will print ALL labels on this printer\t|\n")
			fmt.Fprintf(w, "| help\t| display help text\t|\n")
			fmt.Fprintf(w, "| modify filtername\t| change the text that get's printer on labels\t|\n")
			fmt.Fprintf(w, "| exit\t| save edits and start program\t|\n")
			fmt.Fprintf(w, "+--------------------\t+--------------------------------------------------------------\t+\n")
			w.Flush()
		default:
			fmt.Fprintln(os.Stderr, "Unknown command.  e.g. add newfilter or delete camp")
		} // switch

	} // for

}

func (l *LabelClient) Print(labels []Visitor) (err error) {
	log := l.Log
	log.V(1).Printf("Print. There are %v labels\n", len(labels))
	for _, label := range labels {
		// take the OVL info add label to print queue
		if err := l.AddToLabelQueue(label); err != nil {
			return fmt.Errorf("exporttoglabels error:%v", err)
		}
	}
	if err = l.ProcessLabelQueue(); err != nil {
		return fmt.Errorf("processlabelqueue error:%v", err)
	}
	return nil
}

func (l *LabelClient) CountUSBPrinters() {
	var (
		out []byte
		err error
	)
	log := l.Log
	out, err = exec.Command("lsusb").CombinedOutput()
	if err != nil {
		log.Fatalf("exec.Command() failed error:%v\n", err)
	}
	l.DymoCount = 0
	l.BrotherCount = 0
	log.V(1).Printf("lsusb:%v", string(out))
	lines := strings.Split(string(out), "\n")
	if len(lines) == 0 {
		return
	}
	// count the number of brother or dymo printers
	for _, line := range lines {
		switch {
		case strings.Contains(line, "Brother"):
			l.BrotherCount += 1
		case strings.Contains(line, "LabelWriter"):
			l.DymoCount += 1
		}
	}

}

func (l *LabelClient) updateCUPSPrinter() {
	var (
		brother_count int = l.BrotherCount
		dymo_count    int = l.DymoCount
		p             Printer
		manufacturer  string
		out           []byte
		err           error
		labelByte     []byte
		lines         []string
	)
	log := l.Log
	/*-----------------------------------------------------------------
	 * Execute the lpstat -p command
	 *---------------------------------------------------------------*/
	out, err = exec.Command("lpstat", "-p").CombinedOutput()
	if err != nil {
		log.V(0).Fatalf("exec.Command() failed error:%v\n", err)
	}
	lines = strings.Split(string(out), "\n")
	if len(lines) == 0 {
		log.V(0).Fatalf("No Printers Found")
	}
	l.PrinterQueue = make([]string, 0)
	/*-----------------------------------------------------------------
	 * Process output of the lpstat -p command.  Line by line.
	 *---------------------------------------------------------------*/
loop:
	for _, line := range lines {
		model, _, status, err := l.parseLpstatP(line)
		if err != nil || model == "" {
			continue loop
		}
		log.V(1).Printf("model:%v status:%v\n", model, status)
		/*--------------------------------------------------------------
		 * Filter out the non-label printers, and Only add printers
		 * that are CURRENTLY attached to usb.
		 * NOTE: the count is based on lsusb.  Code may get confused if
		 *       we ever attach different printers from same brand.
		 *------------------------------------------------------------*/
		switch {
		case strings.Contains(model, "QL") && brother_count > 0:
			manufacturer = "BROTHER"
			brother_count -= 1
		case strings.Contains(model, "LabelWriter") && dymo_count > 0:
			manufacturer = "DYMO"
			brother_count -= 1
		default:
			continue loop
		}
		if status == "disabled" {
			log.V(0).Printf("Found disabled label printer.  Attempting to enable:%v\n", model)
			if out, err := exec.Command("lpadmin", "-p", model, "-E").CombinedOutput(); err != nil {
				log.V(0).Printf("lpadmin -p %v -E\n%v\n", model, string(out))
			}
		}
		/*--------------------------------------------------------------
		 * If not exist, create a new printer
		 * -----------------------------------------------------------*/
		var exist bool
		if err := os.Chdir(l.LabelDir); err != nil {
			log.V(0).Printf("Label directory does not exist. path:%v\n", l.LabelDir)
		}
		if p, exist = l.Printers[model]; !exist {
			p.PrinterManufacturer = manufacturer
			p.PrinterModel = model
			p.CurrentJob = ""
			p.JobQueue = make([]Job, 0)
		}
		path := filepath.Join(l.LabelDir, p.PrinterManufacturer+".glabels")
		log.V(1).Printf("Read label Path:%v\n", path)
		labelByte, err = os.ReadFile(path)
		if err != nil {
			log.Fatalf("The label template is missing.  Please create template the program glabels_qt. \nError:%v", err)
		}
		p.LabelTemplate = string(labelByte)
		log.V(2).Printf("template:%v\n", p.LabelTemplate)
		log.V(1).Printf("labeldir:%v file:%v.glabels\n", l.LabelDir, p.PrinterManufacturer)
		/*--------------------------------------------------------------
		 * Update status
		 * -----------------------------------------------------------*/
		p.CurrentTime = time.Time{}
		p.PrinterStatus = status
		l.Printers[model] = p
		l.PrinterQueue = append(l.PrinterQueue, model)
		log.V(1).Printf("printer [%v:%v]\n", p.PrinterManufacturer, p.PrinterModel)
	} // for each line in lpstat -p
}

// issue lpstat -W not-completed, and return results
func (l *LabelClient) GetNotCompleted() (lines []string) {
	var (
		out []byte
		err error
	)
	log := l.Log
	/*------------------------------------------------------------------
	 * Execute lpstat -W not-completed
	 *----------------------------------------------------------------*/
	if out, err = exec.Command("lpstat", "-W", "not-completed").CombinedOutput(); err != nil {
		log.V(0).Printf("lpstat -W all\n  error:%v\n  output:%v\n", err, string(out))
		return
	}
	lines = strings.Split(string(out), "\n")
	return
}

// This works because there is only this program printing on this
// computer.  It grabs the last job in the queue.
func (l *LabelClient) getJobNumber() string {
	log := l.Log
	lines := l.GetNotCompleted()
	if len(lines) < 2 {
		return ""
	}
	// Debug print
	for i, line := range lines {
		log.V(1).Printf("%v lpstat:%v\n", i, line)
	}
	/*------------------------------------------------------------------
	 * The command should have a blank line at the end of the response
	 * So grab the second to the last line.
	 *----------------------------------------------------------------*/
	line := lines[len(lines)-2]
	tokens := strings.Split(line, " ")
	if len(tokens) < 1 {
		return ""
	}
	return tokens[0]
}

func (l *LabelClient) ClearPrintQueue() {
	log := l.Log
	var err error
	jobs := l.GetNotCompleted()
	for _, job := range jobs {
		line := strings.Split(job, " ")
		if len(line) == 0 {
			return
		}
		log.V(1).Printf("Cancel print job:%v\n", line[0])
		if _, err = exec.Command("cancel", line[0]).CombinedOutput(); err != nil {
			log.V(1).Printf("cancel command error:%v\n", err)
		}
	}
}

// EXPERIMENTAL.  Polls lpstat -p till printers are not printing
// For test mostly.  Not sure this works
func (l *LabelClient) WaitTillIdle(seconds int) (isidle bool) {
	log := l.Log
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

		time.Sleep(time.Second)
	}
	return
}

// EXPERIMENTAL. Monitor print queues for all the printers and wait till they are
// empty.  For test.  Not sure this works
func (c *LabelClient) IsStuck(p Printer) bool {
	if !p.CurrentTime.IsZero() && time.Since(p.CurrentTime) > 60*time.Second {
		return true
	}
	return false
}
