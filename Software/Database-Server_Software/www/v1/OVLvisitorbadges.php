<?php

// When called, reply with zero to five visitors needing labels
// and reset their flag to zero in the database
//
// The vrss parameter is required. If missing, return an error.
// vrss is a pipe delimited array of strings. Each string is used
// to select a set of badges that contain one or more of the strings as a
// substring in the visit reason. The strings are case insensitive.
// For example: vrss=tour|class will return badges with visit reasons
// that contain the string "tour" or the word "class".
//
// Creative Commons: Attribution/Share Alike/Non Commercial (cc) 2024 Maker Nexus
// By Jim Schrempp

include 'OVLcommonfunctions.php';


$today = new DateTime();  
$today->setTimeZone(new DateTimeZone("America/Los_Angeles"));
$nowSQL = $today->format("Y-m-d");

$OVLdebug = false; // set to true to see debug messages
debugToUser( "OVLdebug is active. " . $today->format("Y-m-d H:i:s") .  "<br>");


//allowWebAccess();  // if IP not allowed, then die

// Check for the vrss (Visit Reason Sub String) parameter
// This is required.
// We will only return badges to print that have 
if (isset($_GET["vrss"])) {
    $vrss = $_GET["vrss"];
    if (strlen($vrss) == 0) {
        echo( "Error: parameter is empty");
        exit;
    }
} else {
    echo( "Error: parameter is missing");
    exit;
}

// Get the config data
$ini_array = parse_ini_file("OVLconfig.ini", true);
$dbUser = $ini_array["SQL_DB"]["writeUser"];
$dbPassword = $ini_array["SQL_DB"]["writePassword"];
$dbName = $ini_array["SQL_DB"]["dataBaseName"];

$con = mysqli_connect("localhost",$dbUser,$dbPassword,$dbName);

$pathToOVL = $ini_array["OVL"]["pathToOVL"];

// Check connection
if (mysqli_connect_errno()) {
    debugToUser( "Failed to connect to MySQL: " . mysqli_connect_error());
    //logfile("Failed to connect to MySQL: " . mysqli_connect_error());
}


// convert vrss into sql clauses
$vrssArray = explode("|", $vrss);
// for each element in the array, create a clause
$vrssSQL = "(";
foreach ($vrssArray as $vrssElement) {
    $vrssSQL = $vrssSQL . " visitReason LIKE '%" . $vrssElement . "%' OR";
}
$vrssSQL = substr($vrssSQL, 0, -2); // remove trailing OR
$vrssSQL = $vrssSQL . ")";


$visitorArray =  array();  // create empty array here, in case of SQL error
$sql = "SELECT recNum, nameFirst, nameLast, visitReason FROM ovl_visits " 
        . " WHERE labelNeedsPrinting = 1"
        . " AND " . $vrssSQL
        . " LIMIT 5";

debugtouser("sql --> " . $sql . "<br>");

$result = mysqli_query($con, $sql);
if (!$result) {

    debugtouser( "Error: " . $sql . "<br>" . mysqli_error($con));
    //logfile("Error: " . $sql . "<br>" . mysqli_error($con));
    exit;

} else {
    
    $visitorCount = -1;
    $recNumList = "";
    if (mysqli_num_rows($result) > 0) {
        // loop over all rows
        while ($row = mysqli_fetch_assoc($result)) {
            $visitorCount += 1;
            $visitorArray[$visitorCount]["recNum"] = $row["recNum"];
            $visitorArray[$visitorCount]["nameFirst"] = $row["nameFirst"];
            $visitorArray[$visitorCount]["nameLast"] = $row["nameLast"];
            $visitorArray[$visitorCount]["URL"] = $pathToOVL . "OVLcheckinout.php?vid=" . $row["recNum"];
            $visitorArray[$visitorCount]["visitReason"] = $row["visitReason"];
            $recNumList = $recNumList . $row["recNum"] . ",";
        }
        $recNumList = substr($recNumList, 0, -1); // remove trailing comma

        debugToUser( "recNumList: " . $recNumList . "<br>" . strlen($recNumList) . "<br>");

        if (strlen($recNumList) > 0) {

            # update the datatbase to show that the labels have been printed
            $sql = "UPDATE ovl_visits SET labelNeedsPrinting = 0 WHERE recNum in (<<RECNUMLIST>>)";

            $sql = str_replace("<<RECNUMLIST>>", $recNumList, $sql);

            debugtouser("sql" . $sql . "<br>");

            $result = mysqli_query($con, $sql);
            if (!$result) {
                debugtouser( "Error: " . $sql . "<br>" . mysqli_error($con));
                //logfile("Error: " . $sql . "<br>" . mysqli_error($con));
                exit;
            }
        }
    }
}

#convert to json
$arrayForJSON = array();
$arrayForJSON["dateCreated"] =  date("Y-m-d H:i:s");
$arrayForJSON["data"] = array(
        "visitors" => $visitorArray);
$json = json_encode($arrayForJSON);

#send the json
echo $json;
            
// close the database connection
mysqli_close($con);

// end the php
exit;


//-------------------------------------
// Echo a string to the user for debugging
function debugToUser ($data) {
    global $OVLdebug;
    if ($OVLdebug){
        echo "<br>" . $data . "<br>";
    }
}

?>