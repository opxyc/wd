#!/bin/bash
w=$(ls | grep c | wc -l)
if [ $w -lt 5 ]
then
    >&2 echo "less num of files starting with c"
    exit 1
fi