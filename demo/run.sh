#!/bin/bash

if [ ! -f ../serverless-benchmark ]; then
    echo "Error: ../serverless-benchmark does not exist."
    exit 1
fi

# Run the serverless-benchmark binary 10 times
for i in {1..20}
do
    for config in configs/*.json; do
        echo "Run #$i with config $config"
        ../serverless-benchmark --config-path "$config"
        if [ $? -ne 0 ]; then
            echo "serverless-benchmark failed on run #$i with config $config"
            exit 1
        fi
        sleep 3
    done
done

# Generate the report using Jupyter nbconvert
echo "Generating HTML report..."
jupyter nbconvert --to html report.ipynb --output report.html --no-input

echo "Generating PDF report..."
jupyter nbconvert --to pdf report.ipynb --output report.pdf --no-input

echo "All done. Reports generated: report.html and report.pdf"
