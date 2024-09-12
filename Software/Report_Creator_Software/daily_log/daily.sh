#!/bin/bash
f#!/bin/bash
function run_daily_report {
  cd /$HOME/Documents/Online_Visitor_Log/Software/Report_Creator_Software/daily_log
  ./daily_log
}

while true; do
  hour=`date "+%H"`
  echo hour:$hour
  if [[ $hour -ge "1" && $hour -le "8" ]]; then
    echo 'the hour is' $hour
    sleep 60m
  else
    run_daily_report
    sleep 5m
  fi
done
