package main

import (
	"flag"
	"testing"
	"time"
	// "os/exec"

	
	"example.com/debug"
	label "example.com/label"
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
func TestMain(t *testing.T) {
	// init command line flags
	flag.Parse()
	
	log = debug.NewLogClient(*logLevel)
	
	//  Create the label client
	l := label.NewLabelClient(log, *dbURL)
	
	//*********************************************************
	testStart := time.Now()
	print(test1,l)
	if !l.WaitTillIdle(120) {
		t.Errorf("Test1 Failed.  Printer got stuck") 
	}
	t.Logf("Test1 Execution Time:%v\n", time.Since(testStart))
	
	test2 := make([]label.Visitor,0)
	print(test2,l)
	checkStuck(l)
	
	print(test3,l)
	if !l.WaitTillIdle(120) {
		t.Errorf("Test3 Successful.  Timeout") 
	}
	
	
	
	
}

func checkStuck(l *label.LabelClient) {
	for _, p := range l.Printers {
		if l.IsStuck(p) {
			log.V(0).Printf("Attention: Printer is stuck:%v\n",p.PrinterModel)
		}
	}
}

