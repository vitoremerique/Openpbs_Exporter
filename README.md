# OpenPBS Exporter for Prometheus

This project is an Exporter developed to integrate the **OpenPBS** resource manager with **Prometheus**, enabling continuous monitoring of high-performance systems. Through this exporter, OpenPBS monitoring metrics are collected and made available to Prometheus for analysis and visualization, which is essential for a detailed view of system behavior.

## Features

- **Metric Collection**: Extracts metrics directly from OpenPBS, covering CPU usage, memory, job status, node state, among others.
- **Integration with Prometheus**: Provides metrics in a Prometheus-compatible format, facilitating analysis and dashboard creation.
- **Continuous Monitoring**: Enables real-time observation of performance and resource usage, helping to identify bottlenecks and optimize the system.
- **Behavioral Insights**: Supports system behavior and trend analysis, assisting in efficient cluster resource management.

## Requirements

- **OpenPBS**: Cluster resource manager.
- **Prometheus**: Monitoring and alerting tool.
- **Go**: Language used to build and run the exporter.

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/your-username/openpbs-exporter.git
   cd openpbs-exporter


2. Build the project::
   ```bash
   go build -o openpbs_exporter

3. Run the exporter:
   ```bash
   ./openpbs_exporter

4. Configure Prometheus to scrape metrics from the exporter by adding the following configuration to prometheus.yml:
```bash
   scrape_configs:
  - job_name: "openpbs"
    static_configs:
      - targets: ["localhost:porta_exporter"]
```
## Available Metrics

The exporter collects several metrics from OpenPBS, including:

- Node Status: Monitoring the state of individual nodes.
- Job Count: Quantities of pending, running, and completed jobs.
- Resource Usage by User: Detailed CPU and memory statistics by user.
- Cluster Utilization: Aggregated metric of overall cluster usage.

## Grafana Dashboard Structure
This project includes a Grafana dashboard structure in JSON format, which can be used to visualize collected metrics. The JSON structure facilitates the creation of charts, tables, and other visualizations, enabling comprehensive system analysis.
To use the JSON file in Grafana:

1. Import the dashboard JSON file into Grafana.
2. Configure the variables according to the Prometheus environment.
3. Enjoy visual monitoring of OpenPBS metrics directly in Grafana.

## How This Tool Helps with Monitoring
This exporter is an essential tool for those managing clusters with OpenPBS, as it:

- Centralizes monitoring in a reliable system like Prometheus.
- Facilitates dashboard creation in Grafana or other metric visualization tools, enhancing data analysis.
- Helps keep the system under control by detecting potential issues before they impact productivity.
