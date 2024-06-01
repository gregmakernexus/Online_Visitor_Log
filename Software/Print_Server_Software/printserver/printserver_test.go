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
	"testing"
	//"time"

	client "example.com/clientinfo"
	"example.com/debug"
	label "example.com/label"
	// name "github.com/goombaio/namegenerator"
)


// OVL fields recNum,dateCreated,dateCreatedLocal,dateUpdated,dateUpdatedLocal,
//            nameFirst,nameLast,email,phone,visitReason,previousRecNum,dateCheckinLocal,
//            dateCheckoutLocal,elapsedHours,hasSignedWaiver,howDidYouHear,
//            labelNeedsPrinting, notes, okToEmail]
var ovl = []label.Visitor {
  {"firstName":"Kelly","lastName":"Yamanishi","visitReason:":"Forgot Badge"},
  {"firstName":"Greg","lastName":"Yamanishi","visitReason:":"Forgot Badge"},


}




func TestAdd(t *testing.T) {
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
	l := label.NewLabelClient()
		
	// print all the labels return from database
	for _, label := range ovl {
		// take the OVL info and store in .glables file 
		l.ExportToGlabels(label)
		// print the label to the current printer
		printer := fmt.Sprintf("--printer %v", l.Printers[l.Current])
		if out, err := exec.Command("glabels-batch-qt", printer, "temp.glabels").CombinedOutput(); err != nil {
			log.Fatalf("exec.Command failed error:%v\noutput:%v\n", err, out)
		}
		fmt.Printf("Hit enter to print next label>")
		fmt.Scanln()
	} // for each label
}

