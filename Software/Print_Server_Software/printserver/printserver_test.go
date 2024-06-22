package main

import (
	"flag"
	"os"
	"testing"
	// "time"
	"strings"
	// "net/http"
	// "fmt"
	"os/exec"

	"example.com/debug"
	label "example.com/label"
	client "example.com/clientinfo"
	// name "github.com/goombaio/namegenerator"
)

/*---------------------------------------------------------------------
 * OVL fields:
 * [recNum,dateCreated,dateCreatedLocal,dateUpdated,dateUpdatedLocal,
 * nameFirst,nameLast,email,phone,visitReason,previousRecNum,
 * dateCheckinLocal,dateCheckoutLocal,elapsedHours,hasSignedWaiver,
 * howDidYouHear,labelNeedsPrinting, notes, okToEmail]
 *--------------------------------------------------------------------*/ 
var test1 = []label.Visitor{
	{"nameFirst": "Kelly", "nameLast": "Yamanishi", "visitReason": "forgotbadge","URL": "https://makernexus.org"},
	{"nameFirst": "Greg",  "nameLast": "Yamanishi", "visitReason": "forgotbadge","URL": "https://makernexus.org"},
	{"nameFirst": "MyNameisReallyLong",  "nameLast": "lastnameistoolong", "visitReason": "tour","URL": "https://makernexus.org"},
	{"nameFirst": "Kid1",  "nameLast": "smith", "visitReason": "makernexuscamp","URL": "https://makernexus.org"},
	{"nameFirst": "adult",  "nameLast": "moneybags", "visitReason": "makernexusevent","URL": "https://makernexus.org"},
}
var test3 = []label.Visitor{
	{"nameFirst": "Oliver", "nameLast": "Northwood", "visitReason": "forgotbadge","URL": "https://makernexus.org"},
	{"nameFirst": "Cara",  "nameLast": "Stoneburner", "visitReason": "forgotbadge","URL": "https://makernexus.org"},
	{"nameFirst": "alsootfa",  "nameLast": "mohammed", "visitReason": "forgotbadge","URL": "https://makernexus.org"},
	{"nameFirst": "mickey",  "nameLast": "mouse", "visitReason": "forgotbadge","URL": "https://makernexus.org"},
}

var cliCommands = []string{
   "help",
   "",
   "add greg",
   "gregvisit",
   "list",
   "modify greg",
   "hiding",
   "list",
   "delete greg",
   "list",
   "modify greg",
   "invalidcommand",
   "clear",
   "list",
   "reset",
   "list",
//   "clear",
//   "add camp",
   "exit",
}


func TestMain(t *testing.T) {
	// init command line flags
	flag.Parse()

	log = debug.NewLogClient(*logLevel)
	
	
	//  Create the label client
	// rdr  = NewFile(uintptr(syscall.Stdin), "/dev/stdin")
	var rdr *os.File
	var err error
	l := label.NewLabelClient(log, *dbURL)
	
	// load the clientinfo table into map for lookup
	l.Clients, err = client.NewClientInfo(log)
	if err != nil {
		log.V(0).Fatal(err)
	}
	log.V(2).Println("client map:")
	for key, rec := range l.Clients {
		log.V(2).Println(key, rec)
	}
	// test the filtereditor code	
	c := []byte(strings.Join(cliCommands,"\n"))
	os.WriteFile("cli.txt",c, 0777)
	if rdr,err = os.Open("cli.txt"); err != nil {
		t.Error(err)
	    t.Fatal()
	}
	l.FilterEditor(rdr, os.Stdout, true)
	
	
	var labels []label.Visitor
	// clean out any labels that were submitted	
	if labels, err = l.ReadOVL(*dbURL); err != nil {
		log.V(0).Printf("get from webserver failed. Error:%v\n", err)
		return
	}
	for len(labels) != 0 {
		if labels, err = l.ReadOVL(*dbURL); err != nil {
			log.V(0).Printf("get from webserver failed. Error:%v\n", err)
			return
		}
	}
	//*********************************************************
	if err := exec.Command("ovlregister.sh").Run();err != nil {
	   log.V(0).Printf("exec.Command() error:%v\n",err)
	}
    
    for {
		if labels, err = l.ReadOVL(*dbURL); err != nil {
			log.V(0).Printf("get from webserver failed. Error:%v\n", err)
			return
		}
		if len(labels) == 0 {
		  break
		}
		l.Print(labels)
	}
	
}

func checkStuck(l *label.LabelClient) {
	for _, p := range l.Printers {
		if l.IsStuck(p) {
			log.V(0).Printf("Attention: Printer is stuck:%v\n",p.PrinterModel)
		}
	}
}

