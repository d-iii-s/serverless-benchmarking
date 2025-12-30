#!/bin/bash

# Main benchmark script
# First runs measure_first_request.lua to wait for service readiness
# Then runs wrk2 for load testing

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# wrk2 parameters from environment variable
# WRK2PARAMS should contain wrk2 command-line parameters (e.g., "-t2 -c10 -d30s -R1000")
WRK2PARAMS=${WRK2PARAMS:-"-t2 -c10 -d30s -R1000"}
WRK2_SCRIPT="wrk_script.lua"

# Required environment variables
if [ -z "$SERVICE_NAME" ] || [ -z "$PORT" ]; then
    echo -e "${RED}Error: SERVICE_NAME and PORT environment variables must be set${NC}"
    exit 1
fi

BASE_URL="http://${SERVICE_NAME}:${PORT}"

# Set OUTPUT_DIR if not provided (use current directory)
OUTPUT_DIR=${OUTPUT_DIR:-"."}

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Benchmark Runner${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Service: ${BASE_URL}"
echo "Output directory: ${OUTPUT_DIR}"
echo ""

# Step 1: Run measure_first_request.lua
echo -e "${YELLOW}[Step 1/2] Measuring time to first successful request...${NC}"
echo ""

if ! lua measure_first_request.lua; then
    echo -e "${RED}Error: measure_first_request.lua failed${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}✓ Service is ready!${NC}"
echo ""

# Step 2: Run wrk2
echo -e "${YELLOW}[Step 2/2] Running wrk2 load test...${NC}"
echo ""
echo "Parameters: ${WRK2PARAMS}"
echo "Script: ${WRK2_SCRIPT}"
echo ""

# Build wrk2 command
WRK2_CMD="wrk2 ${WRK2PARAMS} --latency"

# Add script if it exists
if [ -f "$WRK2_SCRIPT" ]; then
    WRK2_CMD="${WRK2_CMD} -s ${WRK2_SCRIPT}"
fi

# Add URL
WRK2_CMD="${WRK2_CMD} ${BASE_URL}"

# Create output directory if it doesn't exist
mkdir -p "${OUTPUT_DIR}"

# Generate output filename with timestamp
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
WRK2_OUTPUT_FILE="${OUTPUT_DIR}/wrk2_results_${TIMESTAMP}.txt"

echo "Command: ${WRK2_CMD}"
echo "Results will be saved to: ${WRK2_OUTPUT_FILE}"
echo ""

# Run wrk2 and save output to file (also display to console)
if ! $WRK2_CMD | tee "${WRK2_OUTPUT_FILE}"; then
    echo -e "${RED}Error: wrk2 failed${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}✓ wrk2 results saved to: ${WRK2_OUTPUT_FILE}${NC}"

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Benchmark completed successfully!${NC}"
echo -e "${GREEN}========================================${NC}"

