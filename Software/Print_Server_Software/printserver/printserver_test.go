package main

import (
	"flag"
	"os"
	"testing"

	"os/exec"
	"strings"

	client "example.com/clientinfo"
	"example.com/debug"
	label "example.com/label"
)

/*---------------------------------------------------------------------
 * OVL fields:
 * [recNum,dateCreated,dateCreatedLocal,dateUpdated,dateUpdatedLocal,
 * nameFirst,nameLast,email,phone,visitReason,previousRecNum,
 * dateCheckinLocal,dateCheckoutLocal,elapsedHours,hasSignedWaiver,
 * howDidYouHear,labelNeedsPrinting, notes, okToEmail]
 *--------------------------------------------------------------------*/

func TestMain(t *testing.T) {
	// init command line flags
	flag.Parse()
	// override the database parameter to point at sandbox
	*dbURL = "https://rfidsandbox.makernexuswiki.com/v2/OVLvisitorbadges.php"
	log = debug.NewLogClient(*logLevel)
	//  Create the label client
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
	/*-----------------------------------------------------------
	 * Clean out any labels already submitted, and print them
	 *----------------------------------------------------------*/
	var cliCommands = []string{
		"reset",
		"list",
		"exit",
	}
	/*---------------------------------------------------------------
	 * Test filter code.  Print only camp
	 *-------------------------------------------------------------*/
	c := []byte(strings.Join(cliCommands, "\n"))
	os.WriteFile("cli.txt", c, 0777)
	if rdr, err = os.Open("cli.txt"); err != nil {
		t.Error(err)
		t.Fatal()
	}
	l.FilterEditor(rdr, os.Stdout, true)
	var labels []label.Visitor
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
	/*----------------------------------------------------------
	 * This script cd to test directory, activates the venv, and
	 * runs python ovlregistister.py which executes selenium to register
	 * users
	 *-----------------------------------------------------------*/
	if err := exec.Command("ovlregister.sh").Run(); err != nil {
		log.V(0).Printf("exec.Command() error:%v\n", err)
	}
	/*-----------------------------------------------------------
	 *  Test the filtereditor code
	 *----------------------------------------------------------*/
	cliCommands = []string{
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
		"clear",
		"add camp",
		"modify camp",
		"filter 1 of 1",
		"list",
		"exit",
	}
	/*---------------------------------------------------------------
	 * Test filter code.  Print only camp
	 *-------------------------------------------------------------*/
	c = []byte(strings.Join(cliCommands, "\n"))
	os.WriteFile("cli.txt", c, 0777)
	if rdr, err = os.Open("cli.txt"); err != nil {
		t.Error(err)
		t.Fatal()
	}
	l.FilterEditor(rdr, os.Stdout, true)
	l.ExportTestToGlabels()

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
	/*----------------------------------------------------------------
	 * Print the rest of the labels
	 *--------------------------------------------------------------*/
	cliCommands = []string{
		"reset",
		"list",
		"exit",
	}
	c = []byte(strings.Join(cliCommands, "\n"))
	os.WriteFile("cli.txt", c, 0777)
	if rdr, err = os.Open("cli.txt"); err != nil {
		t.Error(err)
		t.Fatal()
	}
	l.FilterEditor(rdr, os.Stdout, true)
	l.ExportTestToGlabels()
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
