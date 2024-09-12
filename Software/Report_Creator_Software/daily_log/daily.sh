#!/bin/bash
function run_daily_report {
  cd /$HOME/Online_Visitor_Log/Software/Report_Creator_Software/daily_report
  ./daily_report
}

while true; do
  hour=`date "+%H"`
  if [[ $hour >= "01" && $hour < "08" ]]; then
    echo 'the hour is' $hour
    sleep 60m
  else
    run_daily_report
    sleep 5m
  fi
done
