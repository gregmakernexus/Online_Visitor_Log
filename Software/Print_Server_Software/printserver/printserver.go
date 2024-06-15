package main

import (
	"flag"
	"fmt"
	"os"
	"time"
	"os/exec"
	// "time"

	client "example.com/clientinfo"
	"example.com/debug"
	label "example.com/label"
)

/*---------------------------------------------------------------
 * CLI parameters
 *--------------------------------------------------------------*/
var dbURL = flag.String("db", "https://rfid.makernexuswiki.com/v1/OVLvisitorbadges.php", "Database Read URL")
var logLevel = flag.Int("V", 0, "Logging level for debug messages")
var config = flag.Bool("config", false, "Enter filter editor, before starting")
var cliAutomation = flag.String("cli","stdin","File for test automation of cli")
/*---------------------------------------------------------------
 * global variables
 *-------------------------------------------------------------*/
var clients map[string][]string
var log *debug.DebugClient
var l *label.LabelClient
var err error
var rdr *os.File = os.Stdin

func main() {
	// init command line flags
	flag.Parse()
	var err error
	log = debug.NewLogClient(*logLevel)
	
	// load the clientinfo table into map for lookup
	clients, err = client.NewClientInfo(log)
	if err != nil {
		log.V(0).Fatal(err)
	}
	log.V(2).Println("client map:")
	for key, rec := range clients {
		log.V(2).Println(key, rec)
	}
	
	//  Create the label client
	if *cliAutomation != "stdin" {
	    if rdr,err = os.Open(*cliAutomation);err != nil {
			log.V(0).Fatal(err)
		}
	}
	l = label.NewLabelClient(log,*dbURL)
	if *config {
		home, _ := os.UserHomeDir()
		os.Chdir(home)
	    result, _ := exec.Command("pm2", "stop",  "printserver").CombinedOutput()
	    fmt.Println(string(result))
	    l.FilterEditor(os.Stdin, os.Stdout, false)
	    result, _ = exec.Command("pm2", "start",  "printserver").CombinedOutput()
	    fmt.Println(string(result))
	}
	
	// Print program banners
	fmt.Println("Print Server v1.00.00  Initialized.  Hit control-c to exit.")
	
	// Print Test Page
	l.ExportTestToGlabels()

	var labels []label.Visitor
	for i := 1; ; i++ {
	     // http get to get the the new ovl entries to print
		if labels, err = l.ReadOVL(*dbURL); err != nil {
			log.V(0).Printf("get from webserver failed. Error:%v\n", err)
			return
		}
		if len(labels) == 0 {
			for _,p := range l.Printers {
				if l.IsStuck(p) {
					log.V(0).Printf("Printer is stuck:%v\n",err)
				}
			}
			continue
		}
		if err = l.Print(labels);err != nil {
		  log.V(0).Printf("%v\n",err)
		}
		time.Sleep(time.Second)
	} // for infinite loop
}



