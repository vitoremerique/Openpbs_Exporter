package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"regexp"
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
	memUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "openpbs_memory_usage_gb",
		Help: "Total memory usage in the OpenPBS cluster in GB.",
	})
	memAvailable = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "openpbs_memory_available_gb",
		Help: "Total memory available in the OpenPBS cluster in GB.",
	})
	cpuUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "openpbs_cpu_assigned_unit",
		Help: "Total cpu usage in the OpenPBS cluster.",
	})
	cpuAvailable = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "openpbs_cpu_available_unit",
		Help: "Total cpu available in the OpenPBS cluster.",
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
	prometheus.MustRegister(memUsage)
	prometheus.MustRegister(memAvailable)
	prometheus.MustRegister(cpuUsage)
	prometheus.MustRegister(cpuAvailable)
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
	collectMemoryUsage(out)
	collectMemoryAvailable(out)
	collectCPUAssigned(out)
	collectCPUAvailable(out)

	nodesInfo := string(out)
	nodeCount.Set(float64(strings.Count(nodesInfo, "Mom =")))
	nodeAvailable.Set(float64(strings.Count(nodesInfo, "state = free")))
	nodeDown.Set(float64(strings.Count(nodesInfo, "state = down")))
	nodeBusy.Set(float64(strings.Count(nodesInfo, "state = job-busy")))
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

func collectMemoryUsage(output []byte) {
	totalMem := int64(0)

	// Cria um scanner para ler a saída linha por linha
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	// Expressão regular para encontrar linhas com `resources_assigned.mem`
	re := regexp.MustCompile(`resources_assigned\.mem = (\d+)(\w+)`)

	for scanner.Scan() {
		line := scanner.Text()
		match := re.FindStringSubmatch(line)
		if match != nil {
			memValue, err := strconv.ParseInt(match[1], 10, 64)
			if err != nil {
				fmt.Println("Erro ao converter valor de memória:", err)
				continue
			}
			unit := match[2]

			// Converte o valor para GB
			switch unit {
			case "kb":
				totalMem += memValue / (1024 * 1024)
			case "mb":
				totalMem += memValue / 1024
			case "gb":
				totalMem += memValue
			case "tb":
				totalMem += memValue * 1024
			default:
				fmt.Println("Unidade desconhecida:", unit)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Erro ao ler a saída:", err)
	}

	memUsage.Set(float64(totalMem))
}

func collectMemoryAvailable(output []byte) {
	totalMem := int64(0)

	// Cria um scanner para ler a saída linha por linha
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	// Expressão regular para encontrar linhas com `resources_available.mem`
	re := regexp.MustCompile(`resources_available.mem = (\d+)(\w+)`)

	for scanner.Scan() {
		line := scanner.Text()
		match := re.FindStringSubmatch(line)
		if match != nil {
			memValue, err := strconv.ParseInt(match[1], 10, 64)
			if err != nil {
				fmt.Println("Erro ao converter valor de memória:", err)
				continue
			}
			unit := match[2]

			// Converte o valor para GB
			switch unit {
			case "kb":
				totalMem += memValue / (1024 * 1024)
			case "mb":
				totalMem += memValue / 1024
			case "gb":
				totalMem += memValue
			case "tb":
				totalMem += memValue * 1024
			default:
				fmt.Println("Unidade desconhecida:", unit)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Erro ao ler a saída:", err)
	}

	memAvailable.Set(float64(totalMem))
}

func collectCPUAvailable(output []byte) {
	totalCPUs := int64(0)

	// Cria um scanner para ler a saída linha por linha
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	re := regexp.MustCompile(`resources_available\.ncpus = (\d+)`)

	for scanner.Scan() {
		line := scanner.Text()
		match := re.FindStringSubmatch(line)
		if match != nil {
			cpuValue, err := strconv.ParseInt(match[1], 10, 64)
			if err != nil {
				fmt.Println("Erro ao converter valor de CPU:", err)
				continue
			}
			totalCPUs += cpuValue
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Erro ao ler a saída:", err)
	}

	cpuAvailable.Set(float64(totalCPUs))
}

func collectCPUAssigned(output []byte) {
	totalCPUs := int64(0)

	// Cria um scanner para ler a saída linha por linha
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	re := regexp.MustCompile(`resources_assigned\.ncpus = (\d+)`)

	for scanner.Scan() {
		line := scanner.Text()
		match := re.FindStringSubmatch(line)
		if match != nil {
			cpuValue, err := strconv.ParseInt(match[1], 10, 64)
			if err != nil {
				fmt.Println("Erro ao converter valor de CPU:", err)
				continue
			}
			totalCPUs += cpuValue
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Erro ao ler a saída:", err)
	}

	cpuUsage.Set(float64(totalCPUs))
}

func main() {
	go func() {
		for {
			collectMetrics()
			time.Sleep(10 * time.Second) // Collect metrics every 10 seconds
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":8080", nil))
}
