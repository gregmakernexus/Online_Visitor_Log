package main

import (
	"flag"
	// "fmt"
	"os"
	"time"

	// "os/exec"
	// "time"
	"io"
	"path/filepath"

	client "example.com/clientinfo"
	"example.com/debug"
	label "example.com/label"
	"github.com/hajimehoshi/go-mp3"
	"github.com/hajimehoshi/oto"
)

/*---------------------------------------------------------------
 * CLI parameters
 *--------------------------------------------------------------*/
// var dbURL = flag.String("db", "https://rfidsandbox.makernexuswiki.com/v1/OVLindex.html","Database Read URL")
var dbURL = flag.String("db", "https://rfid.makernexuswiki.com/v2/OVLvisitorbadges.php", "Database Read URL")
var logLevel = flag.Int("V", 0, "Logging level for debug messages")

/*---------------------------------------------------------------
 * global variables
 *-------------------------------------------------------------*/
var log *debug.DebugClient
var l *label.LabelClient
var firstLine string

func main() {
	// init command line flags
	flag.Parse()
	var err error

	log = debug.NewLogClient(*logLevel)
	l = label.NewLabelClient(log, *dbURL)

	// load the clientinfo table into map for lookup
	l.Clients, err = client.NewClientInfo(log)
	if err != nil {
		log.V(0).Fatal(err)
	}
	log.V(3).Println("client map:")
	for key, rec := range l.Clients {
		log.V(3).Println(key, rec)
	}

	// Print program banners
	log.V(0).Println("Print Server v2.00.00  Initialized.  Hit control-c to exit.")
	play("Windows_XP_Startup.mp3")
	// Print Test Page
	l.ExportTestToGlabels()
	waitForPrinterOK()

	var labels []label.Visitor

	for i := 1; ; i++ {
		/*--------------------------------------------------------------
		 *  http get to read the the new ovl entries to print
		 *------------------------------------------------------------*/
		if labels, err = l.ReadOVL(*dbURL); err != nil {
			log.V(0).Printf("get from webserver failed. Error:%v\n", err)
			return
		}
		/*-------------------------------------------------------------
		 * Wait here till a printer is plugged in
		 *-----------------------------------------------------------*/
		waitForPrinterToAppear()
		/*-------------------------------------------------------------
		 * if labels to print, add them to the print queue
		 *------------------------------------------------------------*/
		if len(labels) > 0 {
			if err = l.Print(labels); err != nil {
				log.V(0).Printf("%v\n", err)
			}
		}
		/*-------------------------------------------------------------
		 * Wait here while nothing is printing
		 *-----------------------------------------------------------*/
		waitForPrinterOK()
		time.Sleep(time.Second)
	} // for infinite loop
}
func waitForPrinterToAppear() (err error) {
	alarmCount := 0
	alarmSeconds := 0
	l.CountUSBPrinters()
	for l.BrotherCount+l.DymoCount == 0 {
		if alarmCount >= 5 {
			log.Fatalf("Alarm Count Exceeded number:%v\n", alarmCount)
		}
		if alarmCount == 0 || alarmSeconds > 30 {
			log.V(0).Printf("No Label Printers Detected\n")
			play("sound-effect-doorbell-rings-double-200533.mp3")
			alarmSeconds = 0
			alarmCount++
		}
		alarmSeconds++
		time.Sleep(time.Second)
		l.CountUSBPrinters()
	}
	if alarmCount > 0 {
		log.V(0).Printf("Printers. Brother:%v Dymo:%v\n", l.BrotherCount, l.DymoCount)
	}
	return
}
func waitForPrinterOK() (err error) {
	alarmCount := 0
	alarmSeconds := 0
loop:
	for {
		lines := l.GetNotCompleted()
		switch {
		case len(lines) == 0, // nothing to print
			len(lines) == 1 && lines[0] == "":
			alarmSeconds = 0
			firstLine = ""
			break loop
		case firstLine != lines[0]: // new job at the front of queue
			log.V(1).Printf("different job in at head of queue\n")
			alarmCount = 0
			alarmSeconds = 0
			firstLine = lines[0]
			return
		case firstLine == lines[0]: // STUCK!
			log.V(1).Printf("same line at head of queue\n")
			switch {
			// 5 alarms, shutdown
			case alarmCount > 5:
				log.Fatalf("Alarm Count Exceeded number:%v", alarmCount)
			// no alarms and we are stuck for 5 seconds
			case alarmCount == 0 && alarmSeconds > 10:
				log.V(0).Printf("Printer Job Stuck.  Check printer paper. jobs:%v\nlength:%v,%v\n",
					len(lines), len(lines[0]), lines[0])
				play("sound-effect-doorbell-rings-double-200533.mp3")
				alarmCount++
				continue loop
			case alarmCount > 0 && alarmSeconds > 120:
				log.V(0).Printf("Printer Job Stuck.  Check printer paper. jobs:%v\nlength:%v,%v\n",
					len(lines), len(lines[0]), lines[0])
				play("sound-effect-doorbell-rings-double-200533.mp3")
				time.Sleep(time.Duration(10) * time.Second)
				alarmSeconds = 0
				alarmCount++
				continue loop
			}
			time.Sleep(time.Second)
			alarmSeconds++
		}

	} // for
	return
}

func play(audiofile string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Error getting user home directory:%v\n", err)
	}
	path := filepath.Join(home, "Music", audiofile)
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	d, err := mp3.NewDecoder(f)
	if err != nil {
		return err
	}

	c, err := oto.NewContext(d.SampleRate(), 2, 2, 8192)
	if err != nil {
		return err
	}
	defer c.Close()

	p := c.NewPlayer()
	defer p.Close()

	if _, err := io.Copy(p, d); err != nil {
		return err
	}
	return nil
}
