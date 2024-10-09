# sheets-utilities
Utilites to import/export between csv and Google Sheets

The directory structure is as follows:
Online_Visitor_Log
|-- go.work                 directory is a golang workspace.  Defines local packages.
|-- Software
    |- Report_Creator_Software
       |-- debug            debug logging package used by applications                        
       |-- sheet            google sheets client 
       |-- visitor_report   Application to download OVL database and write it to sheet
       |-- waiver_report    Application to read a csv and write it to a google sheet.
       |-- daily_log        Application to download todays OVL entries and write it to sheet 
       |-- install.sh       Bash script that runs the install.sh script for everything

## visitor_report

This application has lots of dependencies.  On of them is the mysql database client written in golang.
For some reason the package is not available for 32-bit Raspbian.  The package must be downloaded
manually.

``` bash
cd /Online_Visitor_Log/Software/Report_Creator_Software
git clone https://github.com/go-sql-driver/mysql.git
cd /Online_Visitor_Log
nano work.go
```
Add a line to the work.go:
```
use ./Online_Visitor_Log/Software/Report_Creator_Software/mysql
```
Save it and exit.

Collect this information to run this program:
1. dbname: name of the database
2. url:    url of the server and port number 
3. id:     user id for the database
4. password: pass for the database
5. Spreadheet: "Visitor Log" 
6. Sheet:      "log"
This information is entered once and stored in a hidden location.  Contact support
if there is a problem.

Build and run:
```
cd /Online_Visitor_Log/Software/Report_Creator_Software/visitor_report
go build visitor_report.go
./visitor_report
```
## waiver_report

There are two pieces to this report:
1. waiverdump    - dumps the list of waivers to a csv.  It is a python application that uses 
                   selenium to dump the list from the web site.
2. waiver_report - uploads the csv to a google sheet.

### waiverdump

The python program is in a github repository.  Read and follow the installation instructions.
https://github.com/gregmakernexus/waiverdump

To run this program, you will need the userid/password to access the Resmark Waiver system.
https://app.resmarksystems.com/login/

The first time the program is run, it will prompt you for this information.  It is stored and used
on subsequent runs.

### waiver_report
Build and run:
```
cd /Online_Visitor_Log/Software/Report_Creator_Software/waiver_report
go build waiver_report.go
./waiver_report
```



