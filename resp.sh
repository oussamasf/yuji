#!/bin/bash

# Check if correct number of arguments are provided
if [ $# -ne 1 ]; then
    echo "Usage: $0 <port> <command>"
    exit 1
fi

# Assign input arguments to variables
command=$1

# Convert command to RESP format
IFS=' ' read -ra cmd_parts <<< "$command"
resp_command="*${#cmd_parts[@]}\r\n"
for part in "${cmd_parts[@]}"; do
    part_length=${#part}
    resp_command+="\$${part_length}\r\n${part}\r\n"
done

echo ${resp_command}


