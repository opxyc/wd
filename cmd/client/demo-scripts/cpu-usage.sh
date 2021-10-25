#!/bin/bash
cutOff="10"
cpuUsage=$(top -bn1 | grep "Cpu(s)" | \
        sed "s/.*, *\([0-9.]*\)%* id.*/\1/" | \
        awk '{print 100 - $1}')
if [ 1 -eq "$(echo "${cpuUsage} > ${cutOff}" | bc)" ]
then
    >&2 echo "current CPU usage is ${cpuUsage}"
    exit 1
fi