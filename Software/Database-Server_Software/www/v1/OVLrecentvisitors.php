<?php

// Purpose: Display the visitors that were in house over the last 5 days
// Author: Jim Schrempp
// Copywrite: 2024 Maker Nexus
// Creative Commons: Attribution/Share Alike/Non Commercial (cc) 2022 Maker Nexus
// By Jim Schrempp
//
//
// Date: 2024-10-02
//

include 'OVLcommonfunctions.php';

$today = new DateTime();  
$today->setTimeZone(new DateTimeZone("America/Los_Angeles"));
$today->add(new DateInterval('P1D'));  // end date for select will be midnight tonight
$nowSQL = $today->format("Y-m-d");

// get date from 5 days ago
$today->sub(new DateInterval('P5D'));
$today->setTime(0,0,0);
$fiveDaysAgoSQL = $today->format("Y-m-d");

$OVLdebug = false; // set to true to see debug messages
debugToUser( "OVLdebug is active. " . $nowSQL .  "<br>");

// allowWebAccess();  // if IP not allowed, then die

// get the HTML skeleton
$html = file_get_contents("OVLrecentvisitors.html");
if (!$html){
  die("unable to open file");
}

// Get the data
$ini_array = parse_ini_file("OVLconfig.ini", true);
$dbUser = $ini_array["SQL_DB"]["readOnlyUser"];
$dbPassword = $ini_array["SQL_DB"]["readOnlyPassword"];
$dbName = $ini_array["SQL_DB"]["dataBaseName"];

$con = mysqli_connect("localhost",$dbUser,$dbPassword,$dbName);

// Check connection
if (mysqli_connect_errno()) {
    echo "Failed to connect to MySQL: " . mysqli_connect_error();
    //logfile("Failed to connect to MySQL: " . mysqli_connect_error());
}

$sql = "SELECT recNum, dateCheckinLocal, nameFirst, nameLast FROM ovl_visits " 
        . " WHERE dateCheckinLocal between '" . $fiveDaysAgoSQL . "'"
        . "         AND '" . $nowSQL . "'"
        . " ORDER BY dateCheckinLocal DESC, nameLast ASC, nameFirst ASC";

$result = mysqli_query($con, $sql);
if (!$result) {

    echo "Error: " . $sql . "<br>" . mysqli_error($con);
    //logfile("Error: " . $sql . "<br>" . mysqli_error($con));
    exit;

} else {

    // create the divs

    $outputTable = "<TABLE>";
    if (mysqli_num_rows($result) == 0) {
        $outputTable = "<tr><td>No visitors in last five days</td></tr>";
    } else {
        // loop over all rows
        while ($row = mysqli_fetch_assoc($result)) {
            //echo "row: " . $row["nameFirst"] . " " . $row["nameLast"] . "<br>";
            $outputTable = $outputTable . makeRow($row["dateCheckinLocal"],$row["nameFirst"], $row["nameLast"]);
        }
    }
    $outputTable = $outputTable . "</TABLE>"; 

    // replace the divs in the html
    $html = str_replace("<<DIVSHERE>>", $outputTable, $html);
    echo $html;
    
}

// close the database connection
mysqli_close($con);

// end the php
die;

// -------------------------------------
// Functions

// make a div
function makeRow($checkinDate, $nameFirst, $nameLast) {
    
    $div = "<tr>"
        . "<td>Date: " . substr($checkinDate,0,10) . "</td>"
        . "<td style='padding:0 20px 0 0'>" . $nameFirst . "  " . $nameLast . "</td>"
        . "</tr>\r\n";
    return $div;
}


// Echo a string to the user for debugging
function debugToUser ($data) {
    global $OVLdebug;
    if ($OVLdebug){
        echo "<br>" . $data . "<br>";
    }
}

?>
