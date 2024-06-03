package main

import (
	"flag"
	"testing"

	
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
var ovl = []label.Visitor{
	{"nameFirst": "Kelly", "nameLast": "Yamanishi", "visitReason": "forgotbadge","URL": "https://makernexus.org"},
	{"nameFirst": "Greg",  "nameLast": "Yamanishi", "visitReason": "forgotbadge","URL": "https://makernexus.org"},
	{"nameFirst": "MyNameisreallylong",  "nameLast": "lastnameistoolong", "visitReason": "tour","URL": "https://makernexus.org"},
}

func TestAdd(t *testing.T) {
	// init command line flags
	flag.Parse()
	
	log = debug.NewLogClient(*logLevel)
	
	//  Create the label client
	l := label.NewLabelClient(log)
	
	print(ovl, l)
	// print all the labels return from database
	//for _, label := range ovl {
		// take the OVL info and store in .glables file
	//	if err = l.ExportToGlabels(label);err != nil {
	//		log.V(0).Printf("ExportToGlabels error%v\n",err)
	//	}
		// print the label to the current printer
	//	printer := fmt.Sprintf("--printer=%v", l.Printers[l.Current].PrinterModel)
	//	log.V(0).Printf("printer:%v\n", printer)
	//	if out, err := exec.Command("glabels-batch-qt", printer, "temp.glabels").CombinedOutput(); err != nil {
	//		log.Fatalf("glabels-batch-qt exec.Command failed error:%v\noutput:%v\n", err, string(out))
	//	}
	//	fmt.Printf("Hit enter to print next label>")
	//	fmt.Scanln()
	//} // for each label
}
