#!/bin/bash

#---------------------------------------------
#  waiverdump.py opens a web browser and goes
#  to the waiversign website and downloads the list
#  of waivers. It goest to the $HOME/Downloads folder
#-----------------------------------------------
function run_waiverdump {
  cd /$HOME/test
  source env/bin/activate
  for n in {1..3}; do
    python waiverdump.py
    RESULT=$?
    if [ $RESULT -eq 0 ]; then
      return 0
    fi
    echo "waiverdump error:" $RESULT " Re-running program retry#" $n
    sleep 1m
  done
  return -1
}
#---------------------------------------------
#  Main loop
#--------------------------------------------
while true; do
  hour=`date "+%H"`
  hour=$(echo "$hour" | sed 's/^0*//')  # get rid of leading zeroes
  # if the hour is 0 then removing leading zeroes creates a null
  if [[ ! -z "$hour" && "$hour" -eq "2" ]]; then
    echo 'running programs at:' $hour
    run_waiverdump
    waiver_report
  fi
  sleep 60m  
done
