<?php

// Purpose: Display the current visitors
// Author: Jim Schrempp
// Copywrite: 2024 Maker Nexus
// Creative Commons: Attribution/Share Alike/Non Commercial (cc) 2022 Maker Nexus
// By Jim Schrempp
//
//
// Date: 2024-10-16
//

include 'OVLcommonfunctions.php';

$today = new DateTime();  
$today->setTimeZone(new DateTimeZone("America/Los_Angeles"));
$nowSQL = $today->format("Y-m-d H:i:s");

$OVLdebug = false; // set to true to see debug messages
debugToUser( "OVLdebug is active. " . $nowSQL .  "<br>");

allowWebAccess();  // if IP not allowed, then die

// get the HTML skeleton
$html = file_get_contents("OVLcurrentprereg.html");
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

$today = new DateTime();  
$today->setTimeZone(new DateTimeZone("America/Los_Angeles"));
$nowSQL = $today->format("Y-m-d"); // just the date

// find all preregistered visitors that have not checked in
$sql = "SELECT recNum, nameFirst, nameLast FROM ovl_visits "
        . " WHERE dateCheckinLocal  = 0"
        . " and recNum not IN"
        . "      (select DISTINCT previousRecNum from ovl_visits) "
        . " ORDER BY nameLast, nameFirst";

$result = mysqli_query($con, $sql);
if (!$result) {

    echo "Error: " . $sql . "<br>" . mysqli_error($con);
    //logfile("Error: " . $sql . "<br>" . mysqli_error($con));
    exit;

} else {

    // create the divs

    $outputDivs = "";
    if (mysqli_num_rows($result) == 0) {
        $outputDivs = "<div class='visitor'>No visitors at this time</div>";
    } else {
        // loop over all rows
        while ($row = mysqli_fetch_assoc($result)) {
            //echo "row: " . $row["nameFirst"] . " " . $row["nameLast"] . "<br>";
            $outputDivs = $outputDivs . makeDiv($row["recNum"],$row["nameFirst"], $row["nameLast"]);
        }
    }

    // replace the divs in the html
    $html = str_replace("<<DIVSHERE>>", $outputDivs, $html);
    echo $html;
    
}

// close the database connection
mysqli_close($con);

// end the php
die;

// -------------------------------------
// Functions

// make a div
function makeDiv($visitID, $nameFirst, $nameLast) {
    
    $div = "<div class='visitor'>"
        . "<a id='reprintbadge' href='OVLreprintbadge.php?vid=" . $visitID . "'>"
        . $nameFirst . " "
        . $nameLast . " "
        . "</a>"
        . "&nbsp;"
        . "</div>\r\n";
    return $div;
}


// make a new badge link
function makeNewBadgeLink($visitID) {
    $link = "<a href='OVLreprintbadge.php?vid=" . $visitID . "'>Reprint Badge</a>";
    return $link;
}

// Echo a string to the user for debugging
function debugToUser ($data) {
    global $OVLdebug;
    if ($OVLdebug){
        echo "<br>" . $data . "<br>";
    }
}

?>
