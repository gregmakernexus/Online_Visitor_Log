<?php

// Purpose: Online Visitor Log Signin Page 
// Author: Jim Schrempp
// Copywrite: 2024 Maker Nexus
// Creative Commons: Attribution/Share Alike/Non Commercial (cc) 2022 Maker Nexus
// By Jim Schrempp
//
// This code will return the Online Visitor Log signin page. It will customize the page
// based on the "type" parameter. 
//    "register" then return a page for someone to register but not create a sign in
//    parameter missing, then return a page for someone to sign in
//
// Date: 2024-03-06
//

include 'OVLcommonfunctions.php';

$today = new DateTime();  
$today->setTimeZone(new DateTimeZone("America/Los_Angeles"));
$nowSQL = $today->format("Y-m-d H:i:s");  // used in SQL statements

$OVLdebug = false; // set to true to see debug messages 
debugToUser( "OVLdebug is active. " . $nowSQL .  "<br>");  

logfile(">>>----- OVLsignin.php called");  

// write php errors to the log file
ini_set('log_errors', 1);
ini_set('error_log', 'OVLlog.txt');

// Don't check for IP address because we need this form to be available to the public
// allowWebAccess();  // if IP not allowed, then this function will die

$registerOnly = FALSE;  // default is to sign in
# is this for register only?
if (isset($_GET["type"])) {
    if ($_GET["type"] == "register") {
        $registerOnly = TRUE;
    }   
}

$html = file_get_contents("OVLcheckinout.html");
if (!$html){
    //logfile("unable to open file");
    die("unable to open file");
}

if ($registerOnly) {
    // register only
    $html = str_replace("<<<PREVIOUSVISITNUM>>>", "-2", $html);
    $html = str_replace("<<<SUBMITBUTTONVALUE>>>", "Register", $html);
} else {
    // checkin
    $html = str_replace("<<<PREVIOUSVISITNUM>>>", "-1", $html);
    $html = str_replace("<<<SUBMITBUTTONVALUE>>>", "Check In", $html);
}

// send the HTML
echo $html;

die();

// ----------------------------------------------

//------------------------------------------------------------------------
// Log message to a rolling log file
//
function logfile($logEntry) {
    // rolling log file set up
    $logFile = 'OVLlog.txt';
    $maxSize = 50000; // Maximum size of the log file in bytes
    $backupFile = 'OVLlog_backup_' . date('Y-m-d') . '.txt'; 

    // Check if the log file is larger than the maximum size
    if (filesize($logFile) > $maxSize) {
        // Rename the log file to the backup file
        rename($logFile, $backupFile);
    }

    // add a carriage return to the log entry
    $logEntry = $logEntry . "\n\r";
    // add a date/time stamp to the log entry
    $logEntry = date('Y-m-d H:i:s') . " " . $logEntry;

    // Write to the log file
    file_put_contents($logFile, $logEntry, FILE_APPEND);
}

// Echo a string to the user for debugging
function debugToUser ($data) {
    global $OVLdebug;
    if ($OVLdebug){
        echo "<br>" . $data . "<br>";
    }
}

?>  
