# Demo

This folder contains sample analysis scripts and a generated HTML and PDF report for the Serverless Benchmark Suite. The demo demonstrates how to process and visualize the raw benchmark data produced by the suite.

## Technologies Used

- **Python 3.10+**
- **Jupyter Notebook** for interactive analysis
- **Pandas** for data manipulation
- **Matplotlib** for plotting and visualization

## How to Run Demo

*You can manualy collect data and run report script against them:*

1. **Install Python dependencies:**

   From the `demo/` directory (venv can be used), run:
   ```
   pip install -r requirements.txt
   ```

2. **Collect data**
   Collect data into the `data/` folder by running the benchmark harness with the configuration files provided in the `configs/` folder.

3. **Gnerate and view reports:**

   - The HTML report is [`report.html`](report.html).
     - To generate report.html execute: `jupyter nbconvert --to html report.ipynb --output report.html --no-input`
   - The PDF report is [`report.pdf`](report.pdf).
     - To generate report.pdf execute: `jupyter nbconvert --to pdf report.ipynb --output report.pdf --no-input`

*Or you can utilize script, which automatically collects data and generates reports:*
 
The script responsible for running the benchmarks and generating the reports is called `run.sh`. This script automates the process of executing the benchmarks multiple times using the configuration files in `demo/configs/`, and then generates both HTML and PDF versions of the analysis report from the Jupyter notebook. If you want to automatically collect data and generate reports, you can use this script by running it from the `demo/` directory.

> **Note:** The script assumes that the benchmark harness binary named `serverless-benchmark` is present in the parent directory of `demo/`.

> **Note:** You can also test the reporting script using the provided zipped data archive (`data.zip`).

## Data Organization

- **Requirements file:**  
  `requirements.txt` lists all Python dependencies needed to run the analysis and generate the reports.
- **Pre-collected data archive:**  
  `data.zip` contains already measured data on which the report is generated.
- **Configs:**  
  Benchmark configuration files are in `report/configs/`.
- **Analysis scripts:**  
  `report.ipynb` (Jupyter notebook version)
- **HTML report:**  
  `report.html` (generated output)
- **PDF report:**  
  `report.pdf` (generated output)
- **Benchmark and report automation script:**  
  `run.sh` automates running all benchmarks using the configs, then generates both HTML and PDF reports from the notebook.

## Example Workflow

1. **Run benchmarks using the harness and example configs.**
2. **Collect results in `demo/data/`.**
3. **Run the analysis script to generate plots and tables.**
4. **Open `report.html` or `report.pdf` to view the results.**

## Notes

- The provided scripts are examples; you can adapt them for your own analysis needs.
- For more details on the data format and available metrics, see comments in `report.ipynb`.
- If you encounter issues, ensure all dependencies from `requirements.txt` are installed and that your Python version is compatible.

For further details on the benchmark suite, see the main [README](../README.md).