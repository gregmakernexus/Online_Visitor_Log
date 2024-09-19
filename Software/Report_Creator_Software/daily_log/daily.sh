#!/bin/bash
f#!/bin/bash
function run_daily_report {
  cd /$HOME/Documents/Online_Visitor_Log/Software/Report_Creator_Software/daily_log
  ./daily_log
}

while true; do
  hour=`date "+%H"`
  hour=$(echo "$hour" | sed 's/^0*//')
  if [[ $hour -ge "1" && $hour -le "8" ]]; then
    sleep 60m
  else
    run_daily_report
    sleep 5m
  fi
done
