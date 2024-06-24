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
	jobStates = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "openpbs_job_states",
		Help: "Number of jobs by state",
	}, []string{"state"})
	nodeStates = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "openpbs_node_states",
		Help: "Number of nodes by state",
	}, []string{"state"})
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
)

func init() {
	prometheus.MustRegister(jobCount)
	prometheus.MustRegister(nodeCount)
	prometheus.MustRegister(nodeAvailable)
	prometheus.MustRegister(nodeDown)
	prometheus.MustRegister(jobStates)
	prometheus.MustRegister(nodeStates)
	prometheus.MustRegister(nodeBusy)
	prometheus.MustRegister(nodeReserved)
	prometheus.MustRegister(nodeOffline)
	prometheus.MustRegister(nodeDrained)
	prometheus.MustRegister(nodeunknown)
}

func collectMetrics() {
	// Collect job count
	out, err := exec.Command("bash", "-c", "qstat | wc -l").Output()
	if err != nil {
		log.Printf("Error collecting job count: %v", err)
		return
	}
	jobCount.Set(parseOutput(out) - 5) // Subtract header lines

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
	parseJobStates(out)

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

func parseJobStates(output []byte) {
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 3 {
			count, err := strconv.Atoi(fields[0])
			if err == nil {
				user := fields[2]
				jobStates.WithLabelValues(fields[1], user).Set(float64(count))
			}
		}
	}
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
