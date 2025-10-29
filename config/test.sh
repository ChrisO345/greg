#!/usr/bin/env bash

# Get current date and time
datevar=$(date +"%Y-%m-%d")
timevar=$(date +"%H:%M:%S")

# Array of items to output
declare -A items=(
  ["$datevar"]="echo 'Chosen Date'"
  ["$timevar"]="echo 'Chosen Time'"
)

# Output TOML
echo ""
for label in "${!items[@]}"; do
    echo "[[items]]"
    echo "label = \"$label\""
    echo "exec = \"${items[$label]}\""
    echo "visible = true"
    echo ""
done
