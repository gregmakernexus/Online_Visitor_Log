package main

import (
	// "crypto/tls"
	// "encoding/json"
	"flag"
	"fmt"
	//"io"
	//"net/http"
	//"os"
	"os/exec"
	// "path/filepath"
	// "strconv"
	//"strings"
	"time"

	client "example.com/clientinfo"
	"example.com/debug"
	label "example.com/label"
	// name "github.com/goombaio/namegenerator"
)
/*---------------------------------------------------------------
 * CLI parameters
 *--------------------------------------------------------------*/
var dbURL = flag.String("db", "https://rfid.makernexuswiki.com/v1/OVLvisitorbadges.php", "Database Read URL")
var printDelay = flag.Int("delay", 0, "Delay between print commands")
var logLevel = flag.Int("V", 0, "Logging level for debug messages")
var clearCache = flag.Bool("clear", false, "Clear database information cache")
/*---------------------------------------------------------------
 * global variables
 *-------------------------------------------------------------*/
var clients map[string][]string
var log *debug.DebugClient
var l *label.LabelClient

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
	l = label.NewLabelClient(log)
	// Print program banners
	fmt.Println("Print Server v1.00.00  Initialized.  Hit control-c to exit.")
	// fmt.Println("Label Print Delay is:", *printDelay)
	// Print Test Page
	l.ExportTestToGlabels()

	var labels []label.Visitor
	for i := 1; ; i++ {
	     // http get to get the the new ovl entries to print
		if labels, err = l.ReadOVL(*dbURL); err != nil {
			log.V(0).Printf("get from webserver failed. Error:%v\n", err)
			return
		}
		// if there are labels, then print
		if len(labels) > 0 {
			if err = print(labels, l); err != nil {
		      log.V(0).Printf("%v\n", err)
			}
		}
		time.Sleep(time.Second * time.Duration(*printDelay))
	} // for infinite loop
}

func print(labels []label.Visitor, l *label.LabelClient) error {
	// print all the labels return from database
	for _, label := range labels {
		// take the OVL info and store in .glables file 
		p,err := l.ExportToGlabels(label)
		if err != nil {
			return fmt.Errorf("exporttoglabels error:%v",err)
		}
		// print the label to the current printer
		// printer := fmt.Sprintf("--printer %v", p.PrinterModel)
		if out, err := exec.Command("glabels-batch-qt", "--printer="+p.PrinterModel, "temp.glabels").CombinedOutput(); err != nil {
			return fmt.Errorf("glabels-batch-qt --printer=%v temp.glabels\n  error:%v\n  output:%v\n", p.PrinterModel, err, out)
		}
	} // for each label
	return nil
}	
