package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	name "github.com/goombaio/namegenerator"
)

var dbURL = flag.String("db", "https://rfidsandbox.makernexuswiki.com/testVisitorLabels.php", "Database Read URL")
var test = flag.Bool("test", false, "Label test to print labels with random names")
var printDelay = flag.Int("delay", 0, "Delay between print commands")
var printFilter = flag.String("filter", "a-z", "Filter on last name. eg. a-f")
var toMonth = []string{"", "Jan.", "Feb.", "Mar.", "Apr.", "Jun.", "Jul.", "Aug.", "Sep.", "Oct.", "Nov.", "Dec."}

type visitor map[string]any
type visitorList struct {
	Create string `json:"dateCreated"`
	Data   struct {
		Visitors []visitor `json:"visitors"`
	} `json:"data"`
}

type labelClient struct {
	labelDir string
	// labelPath     string
	labelTemplate string
	connection    *http.Client
}

func newLabelClient() *labelClient {
	c := new(labelClient)
	var labelByte []byte
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Error getting user home directory:%v\n", err)
	}
	// Read the label file into a string
	c.labelDir = filepath.Join(home, "Mylabels")
	if err := os.Chdir(c.labelDir); err != nil {
		log.Fatal("Label directory does not exist.")
	}
	labelByte, err = os.ReadFile("template.glabels")
	if err != nil {
		log.Printf("The label template is missing.  Please create template the program glabels_qt. \nError:%v", err)
		log.Fatalf("Store it at:%v\n", filepath.Join(c.labelDir, "template.glabels"))

	}
	c.labelTemplate = string(labelByte)

	// Create the http client with no security
	c.connection = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	return c
}

func (c *labelClient) dbRead(url string) ([]visitor, error) {
	// Do the http.get
	labels := new(visitorList)
	html, err := c.connection.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	// extract the body from the http packet
	results, err := io.ReadAll(html.Body)
	if err != nil {
		log.Fatal(err)
	}
	html.Body.Close()
	// unmarshall the json object
	if err := json.Unmarshal(results, labels); err != nil {
		return nil, err
	}
	return labels.Data.Visitors, nil

}
func (c *labelClient) print(info visitor) error {
	var temp string = c.labelTemplate
	// Get the date right now and update the label
	t := time.Now()
	nowDate := fmt.Sprintf("%v %v, %v", toMonth[t.Month()], t.Day(), t.Year())
	temp = strings.Replace(temp, "${Date}", nowDate, -1)
	// loop through each field and update the label
	for key, data := range info {
		log.Println("key:", key, " data:", data)
		dataType := fmt.Sprintf("%T", data)
		switch dataType {
		case "string":
			temp = strings.Replace(temp, "${"+key+"}", data.(string), -1)
		case "int":
			temp = strings.Replace(temp, "${"+key+"}", strconv.Itoa(data.(int)), -1)
		}
	}
	// cd to the Mylabel directory so we can write files
	if err := os.Chdir(c.labelDir); err != nil {
		log.Fatal("Label directory does not exist.")
	}
	// delete and write the temp.glables file
	os.Remove("temp.glabels")
	if err := os.WriteFile("temp.glabels", []byte(temp), 0666); err != nil {
		log.Fatalf("Error writing label file error:%v\n", err)
	}
	// print the label
	cmd := exec.Command("glabels-batch-qt", "temp.glabels")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("cmd.Run() failed error:%v\n", err)
	}

	fmt.Printf("combined out:\n%s\n", string(out))
	return nil
}

func (c *labelClient) printTestPage() error {
	var temp string = c.labelTemplate
	// Get the date right now and update the label
	t := time.Now()
	nowDate := fmt.Sprintf("%v %v, %v", toMonth[t.Month()], t.Day(), t.Year())
	temp = strings.Replace(temp, "${Date}", nowDate, -1)
	temp = strings.Replace(temp, "${nameFirst}", "WELCOME TO MAKERNEXUS", -1)
	temp = strings.Replace(temp, "${nameLast}", "", -1)
	temp = strings.Replace(temp, "Visitor", "Test Label", -1)
	// cd to the Mylabel directory so we can write files
	if err := os.Chdir(c.labelDir); err != nil {
		log.Fatal("Label directory does not exist.")
	}
	// delete and write the temp.glables file
	os.Remove("temp.glabels")
	if err := os.WriteFile("temp.glabels", []byte(temp), 0666); err != nil {
		log.Fatalf("Error writing label file error:%v\n", err)
	}
	// print the label
	cmd := exec.Command("glabels-batch-qt", "temp.glabels")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("cmd.Run() failed error:%v\n", err)
	}

	fmt.Printf("combined out:\n%s\n", string(out))
	return nil
}

func newFilter(filter string) (map[string]int, error) {
	alpha := "abcdefghijklmnopqurstuvwxyz"
	filter = strings.ReplaceAll(filter, " ", "")
	filterSplit := strings.Split(filter, "-")
	if len(filterSplit) != 2 && len(filterSplit[0]) != 1 && len(filterSplit[1]) != 1 {
		return nil, fmt.Errorf("empty or invalid filter:%v", filter)
	}
	var low, high int
	if low = strings.Index(alpha, strings.ToLower(filterSplit[0])); low == -1 {
		return nil, fmt.Errorf("low filter must be alpha:%v", filter)
	}
	if high = strings.Index(alpha, strings.ToLower(filterSplit[1])); high == -1 {
		return nil, fmt.Errorf("high filter must be alpha:%v", filter)
	}

	temp := min(low, high)
	high = max(low, high)
	low = temp

	filterMap := make(map[string]int)
	for i := low; i < high+1; i++ {
		filterMap[string(alpha[i])] = 0
	}
	return filterMap, nil
}

func okToPrint(filterMap map[string]int, lastName string) bool {
	firstChar := lastName[0:1]
	if _, exists := filterMap[firstChar]; !exists {
		fmt.Printf("skipping label.  lastName:%v filtermap:%v\n", lastName, filterMap)
		return false
	}
	return true
}	
func main() {
	// init command line flags
	flag.Parse()
	//  Create the label client
	c := newLabelClient()
	// Print program banners
	fmt.Println("Print Server v1.00.00  Initialized.  Hit control-c to exit.")
	// fmt.Println("Label Print Delay is:", *printDelay)
	// Print Test Page
	c.printTestPage()

	var err error
	var labels []visitor
	// Generate the filter for the last name.
	var filterMap map[string]int
	if filterMap, err = newFilter(*printFilter); err != nil {
		fmt.Printf("Error generating filter map:%v error:%v\n", filterMap, err)
		log.Fatal(3)
	}
	if *test {
		seed := time.Now().UTC().UnixNano()
    	n := name.NewNameGenerator(seed)
		for i := 1; ; i++ {
			randomName := n.Generate()
			n := strings.Split(randomName,"-")
			label := make(visitor)
			if okToPrint(filterMap,n[1]) {
				label["nameLast"] = n[1]
				label["nameFirst"] = n[0]
				label["URL"] = "https://makernexus.org;laskdfjas;ldkfjas;ldfkjas;ldfkjaasdf"
				c.print(label)
			}	
			time.Sleep(time.Second * time.Duration(*printDelay))
		}
	}
	for i := 1; ; i++ {
		// read databse to see if there are labels to print
		if labels, err = c.dbRead(*dbURL); err != nil {
			return
		}
		// if there are no labels then print a dot and continue
		if len(labels) == 0 {
			fmt.Printf("%v", ".")
			time.Sleep(time.Second)
			continue
		}
		for _, label := range labels {
			// Filter label prints by last name
			lastName := label["nameLast"].(string)
			if okToPrint(filterMap, lastName) {
				c.print(label)
			}
			if *printDelay == -1 {
				fmt.Printf("Hit enter to print next label>")
				fmt.Scanln()
			} else {
				fmt.Println("Label Print Delay is:", *printDelay)
				time.Sleep(time.Second * time.Duration(*printDelay))
			}
		}

	}
}

