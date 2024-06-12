package main

import (
	"flag"
	"fmt"
	"time"

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
var clearCache = flag.Bool("clear", false, "Clear database information cache")
/*---------------------------------------------------------------
 * global variables
 *-------------------------------------------------------------*/
var clients map[string][]string
var log *debug.DebugClient
var l *label.LabelClient
var err error

func main() {
	// init command line flags
	flag.Parse()
	var err error
	log = debug.NewLogClient(*logLevel)
	// delete config file and re-input DB data
	if *clearCache {
		client.ClearConfig()
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
	
	//  Create the label client
	l = label.NewLabelClient(log,*dbURL)
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
		if err = print(labels,l);err != nil {
		  log.V(0).Printf("%v\n",err)
		}
		time.Sleep(time.Second)
	} // for infinite loop
}

func print(labels []label.Visitor, l *label.LabelClient) error {
	// log.V(1).Printf("There are %v labels\n",len(labels))
	for _, label := range labels {
		// take the OVL info add label to print queue 
		if err := l.ExportToGlabels(label); err != nil {
			return fmt.Errorf("exporttoglabels error:%v",err)
		}
	}
	if err = l.ProcessLabelQueue(); err != nil {
		return fmt.Errorf("processlabelqueue error:%v", err)
	}
	return nil
}
	

