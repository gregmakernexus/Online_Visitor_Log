package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

var port serial.Port
var buff []byte

func waitForScanner() string {
	for {
		port, err := findScanner()
		if err != nil {
			if strings.Contains(err.Error(), "found") {
				time.Sleep(time.Second * 5)
				continue
			}
		}
		if port != "" {
			return port
		}
	}
}

func findScanner() (string, error) {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return "", err
	}
	if len(ports) == 0 {
		return "", fmt.Errorf("no serial ports found")
	}
	for _, port := range ports {
		if port.IsUSB && strings.Contains(port.Product, "USB Serial Device") {
			fmt.Printf("Found port: %s\n", port.Name)
			fmt.Printf("   Product:%v\n", port.Product)
			fmt.Printf("   USB ID     %s:%s\n", port.VID, port.PID)
			fmt.Printf("   USB serial %s\n", port.SerialNumber)
			return port.Name, nil
		}
	}
	return "", fmt.Errorf("USB serial port not found")
}
func read() (string, error) {
	// Read and print the response
	var qrCode string
	var err error
	var n int
	for {
		// Reads up to 100 bytes
		n, err = port.Read(buff)
		if err != nil {
			return "", err
		}
		if n == 0 {
			fmt.Println("\nEOF")
			break
		}
		qrCode = qrCode + string(buff[:n])
		// If we receive a newline stop reading
		if buff[n-1] == 13 {
			qrCode = qrCode[:n]
			break
		}
	}
	return qrCode, nil

}
func main() {
	mode := &serial.Mode{
		BaudRate: 9600,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}
	var err error
	var scanCount int
	for {
		/*-------------------------------------------------------------
		 *  Wait for a scanner to be plugged in.  Waits forever.
		 *------------------------------------------------------------*/
		comPort := waitForScanner()
		/*--------------------------------------------------------------
		 * Open the scanner
		 *-------------------------------------------------------------*/
		port, err = serial.Open(comPort, mode)
		if err != nil {
			log.Fatal(err)
		}
		/*--------------------------------------------------------------
		 * Main scanner read loop
		 *-------------------------------------------------------------*/
		buff = make([]byte, 100)
		var resp *http.Response
		for {
			qrCode, err := read()
			if err != nil {
				fmt.Printf("%v\nWaiting for scanner to be available\n", err)
				break
			}
			if resp, err = http.Get(qrCode); err != nil {
				fmt.Printf("http error:%v\n", err)
			}
			scanCount++
			now := time.Now()
			fmt.Printf("%5v %v %v status: %d\n", scanCount, now.Format("2006-01-02 15:04:05"), qrCode, resp.StatusCode)

		} // for forever
	} // for forever
}
