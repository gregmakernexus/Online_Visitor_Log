package main

import (
	"flag"
	"os"

	"example.com/debug"
	label "example.com/label"
)

/*---------------------------------------------------------------
 * CLI parameters
 *--------------------------------------------------------------*/
var dbURL = flag.String("db", "https://rfid.makernexuswiki.com/v1/OVLvisitorbadges.php", "Database Read URL")
var logLevel = flag.Int("V", 0, "Logging level for debug messages")
/*---------------------------------------------------------------
 * global variables
 *-------------------------------------------------------------*/
var log *debug.DebugClient
var l *label.LabelClient

func main() {
    log = debug.NewLogClient(*logLevel)
	//  Create the label client
	l = label.NewLabelClient(log,*dbURL)
	l.FilterEditor(os.Stdin,os.Stdout, false)
	
}


