#!/bin/bash

#---------------------------------------------
#  waiverdump.py opens a web browser and goes
#  to the waiversign website and downloads the list
#  of waivers. It goest to the $HOME/Downloads folder
#-----------------------------------------------
function run_waiverdump {
  cd /$HOME/test
  source env/bin/activate
  python waiverdump.py
}
#---------------------------------------------
#  Run reports once at start
#--------------------------------------------
visitor_report
run_waiverdump
waiver_report
#---------------------------------------------
#  Main loop
#--------------------------------------------
while true; do
  hour=`date "+%H"`
  hour=$(echo "$hour" | sed 's/^0*//')  # get rid of leading zeroes
  if [ $hour -eq "2" ]; then
    echo 'running programs at:' $hour
    ./visitor_report
    run_waiverdump
    ./waiver_report
  fi
  if [[ $hour -ge "1" && $hour -le "8" ]]; then
    echo 'Makernexus is closed.  hour:' $hour
    sleep 60m
    continue
  fi

  daily_log
  sleep 5m
  
done
