package main

import (
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	jobCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "openpbs_job_count",
		Help: "Number of jobs in the queue",
	})
	nodeCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "openpbs_node_count",
		Help: "Number of nodes",
	})
	nodeAvailable = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "openpbs_node_available",
		Help: "Number of available nodes",
	})
	nodeDown = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "openpbs_node_down",
		Help: "Number of down nodes",
	})

	nodeBusy = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "openpbs_node_busy",
		Help: "Number of busy nodes",
	})
	nodeReserved = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "openpbs_node_reserved",
		Help: "Number of reserved nodes",
	})
	nodeOffline = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "openpbs_node_offline",
		Help: "Number of offline nodes",
	})
	nodeDrained = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "openpbs_node_drained",
		Help: "Number of drained nodes",
	})
	nodeunknown = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "openpbs_node_unknown",
		Help: "Number of unknown nodes",
	})
	jobRunning = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "openpbs_job_running",
		Help: "Number of jobs in the 'Running' state",
	})
	jobQueued = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "openpbs_job_queued",
		Help: "Number of jobs in the 'Queued' state",
	})
	jobHeld = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "openpbs_job_held",
		Help: "Number of jobs in the 'Held' state",
	})
	jobExiting = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "openpbs_job_exiting",
		Help: "Number of jobs in the 'Exiting' state",
	})
)

func init() {
	prometheus.MustRegister(jobCount)
	prometheus.MustRegister(nodeCount)
	prometheus.MustRegister(nodeAvailable)
	prometheus.MustRegister(nodeDown)
	prometheus.MustRegister(nodeBusy)
	prometheus.MustRegister(nodeReserved)
	prometheus.MustRegister(nodeOffline)
	prometheus.MustRegister(nodeDrained)
	prometheus.MustRegister(nodeunknown)
	prometheus.MustRegister(jobRunning)
	prometheus.MustRegister(jobQueued)
	prometheus.MustRegister(jobHeld)
	prometheus.MustRegister(jobExiting)
}

func collectMetrics() {
	// Collect job count
	out, err := exec.Command("bash", "-c", "qstat | wc -l").Output()
	if err != nil {
		log.Printf("Error collecting job count: %v", err)
		return
	}
	jobCount.Set(parseOutput(out) - 2) // Subtract header lines

	// Collect node information
	out, err = exec.Command("bash", "-c", "pbsnodes -a").Output()
	if err != nil {
		log.Printf("Error collecting node information: %v", err)
		return
	}
	nodesInfo := string(out)
	nodeCount.Set(float64(strings.Count(nodesInfo, "Mom =")))
	nodeAvailable.Set(float64(strings.Count(nodesInfo, "state = free")))
	nodeDown.Set(float64(strings.Count(nodesInfo, "state = down")))
	nodeBusy.Set(float64(strings.Count(nodesInfo, "state = busy")))
	nodeReserved.Set(float64(strings.Count(nodesInfo, "state = reserved")))
	nodeOffline.Set(float64(strings.Count(nodesInfo, "state = offline")))
	nodeDrained.Set(float64(strings.Count(nodesInfo, "state = draining")))
	nodeunknown.Set(float64(strings.Count(nodesInfo, "state = state-unknown,down")))
	// Collect job states
	out, err = exec.Command("bash", "-c", "qstat -a | tail -n +6 | awk '{print $10}' | sort | uniq -c").Output()
	if err != nil {
		log.Printf("Error collecting job states: %v", err)
		return
	}
	parseJobStatesCountperStatus(out)

	// Collect node states
	out, err = exec.Command("bash", "-c", "pbsnodes -a | grep 'state =' | sort | uniq -c").Output()
	if err != nil {
		log.Printf("Error collecting node states: %v", err)
		return
	}
	parseNodeStates(out)

}

func parseOutput(output []byte) float64 {
	result := strings.TrimSpace(string(output))
	count, err := strconv.Atoi(result)
	if err != nil {
		log.Printf("Error parsing output: %v", err)
		return 0
	}
	return float64(count)
}

func parseNodeStates(output []byte) {
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 3 {
			count, err := strconv.Atoi(fields[0])
			if err == nil {
				nodeStates.WithLabelValues(fields[2]).Set(float64(count))
			}
		}
	}
}

func parseJobStatesCountperStatus(output []byte) {
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 1 {
			count, err := strconv.Atoi(fields[0])
			if err == nil {
				state := fields[1]

				// Atualizar os gauges correspondentes ao estado do job
				switch state {
				case "R":
					jobRunning.Set(float64(count))
				case "Q":
					jobQueued.Set(float64(count))
				case "H":
					jobHeld.Set(float64(count))
				case "E":
					jobExiting.Set(float64(count))

				}
			}
		}
	}
}

func main() {
	go func() {
		for {
			collectMetrics()
			time.Sleep(5 * time.Second) // Collect metrics every 60 seconds
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":8080", nil))
}
