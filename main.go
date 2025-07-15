package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	modemHost  = flag.String("modem-host", "https://192.168.100.1", "Hitron CODA56 modem host URL")
	listenAddr = flag.String("listen-addr", ":8080", "Address to listen on for HTTP requests")
	interval   = flag.Duration("interval", 30*time.Second, "Polling interval")
	timeout    = flag.Duration("timeout", 10*time.Second, "HTTP request timeout")
)

type ModemClient struct {
	baseURL string
	client  *http.Client
}

type DownstreamInfo struct {
	ChannelID      int
	Frequency      float64
	PowerLevel     float64
	SNR            float64
	Modulation     string
	Corrected      int64
	Uncorrectables int64
}

type UpstreamInfo struct {
	ChannelID  int
	Frequency  float64
	PowerLevel float64
	SymbolRate float64
	Modulation string
}

type SystemInfo struct {
	UpTime          string
	SystemTime      string
	HardwareVersion string
	SoftwareVersion string
	SerialNumber    string
}

func NewModemClient(baseURL string, timeout time.Duration) *ModemClient {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	return &ModemClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout:   timeout,
			Transport: tr,
		},
	}
}

func (m *ModemClient) get(endpoint string) ([]byte, error) {
	url := fmt.Sprintf("%s/data/%s", m.baseURL, endpoint)
	log.Printf("Requesting: %s", url)

	resp, err := m.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, endpoint)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body for %s: %w", endpoint, err)
	}

	return body, nil
}

func (m *ModemClient) parseDownstreamInfo(data []byte) ([]DownstreamInfo, error) {
	// Parse the HTML/JavaScript response from dsinfo.asp
	var channels []DownstreamInfo

	// This is a simplified parser - in reality, you'd need to parse the actual HTML/JS
	// For now, return empty slice to establish the structure
	log.Println("Parsing downstream info")
	return channels, nil
}

func (m *ModemClient) parseUpstreamInfo(data []byte) ([]UpstreamInfo, error) {
	// Parse the HTML/JavaScript response from usinfo.asp
	var channels []UpstreamInfo

	// This is a simplified parser - in reality, you'd need to parse the actual HTML/JS
	// For now, return empty slice to establish the structure
	log.Println("Parsing upstream info")
	return channels, nil
}

func (m *ModemClient) parseSystemInfo(data []byte) (*SystemInfo, error) {
	// Parse the HTML/JavaScript response from getSysInfo.asp
	var sysInfo SystemInfo

	// This is a simplified parser - in reality, you'd need to parse the actual HTML/JS
	// For now, return empty struct to establish the structure
	log.Println("Parsing system info")
	return &sysInfo, nil
}

func (m *ModemClient) GetDownstreamInfo() ([]DownstreamInfo, error) {
	data, err := m.get("dsinfo.asp")
	if err != nil {
		return nil, err
	}
	return m.parseDownstreamInfo(data)
}

func (m *ModemClient) GetUpstreamInfo() ([]UpstreamInfo, error) {
	data, err := m.get("usinfo.asp")
	if err != nil {
		return nil, err
	}
	return m.parseUpstreamInfo(data)
}

func (m *ModemClient) GetSystemInfo() (*SystemInfo, error) {
	data, err := m.get("getSysInfo.asp")
	if err != nil {
		return nil, err
	}
	return m.parseSystemInfo(data)
}

type MetricsCollector struct {
	client *ModemClient

	// Downstream metrics
	downstreamPower          *prometheus.GaugeVec
	downstreamSNR            *prometheus.GaugeVec
	downstreamFreq           *prometheus.GaugeVec
	downstreamCorrectables   *prometheus.CounterVec
	downstreamUncorrectables *prometheus.CounterVec

	// Upstream metrics
	upstreamPower      *prometheus.GaugeVec
	upstreamFreq       *prometheus.GaugeVec
	upstreamSymbolRate *prometheus.GaugeVec

	// System metrics
	systemInfo *prometheus.GaugeVec
}

func NewMetricsCollector(client *ModemClient) *MetricsCollector {
	return &MetricsCollector{
		client: client,

		downstreamPower: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hitron_downstream_power_dbmv",
				Help: "Downstream channel power level in dBmV",
			},
			[]string{"channel_id", "frequency", "modulation"},
		),

		downstreamSNR: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hitron_downstream_snr_db",
				Help: "Downstream channel signal-to-noise ratio in dB",
			},
			[]string{"channel_id", "frequency", "modulation"},
		),

		downstreamFreq: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hitron_downstream_frequency_hz",
				Help: "Downstream channel frequency in Hz",
			},
			[]string{"channel_id", "modulation"},
		),

		downstreamCorrectables: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "hitron_downstream_correctables_total",
				Help: "Total number of correctable errors on downstream channel",
			},
			[]string{"channel_id", "frequency", "modulation"},
		),

		downstreamUncorrectables: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "hitron_downstream_uncorrectables_total",
				Help: "Total number of uncorrectable errors on downstream channel",
			},
			[]string{"channel_id", "frequency", "modulation"},
		),

		upstreamPower: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hitron_upstream_power_dbmv",
				Help: "Upstream channel power level in dBmV",
			},
			[]string{"channel_id", "frequency", "modulation"},
		),

		upstreamFreq: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hitron_upstream_frequency_hz",
				Help: "Upstream channel frequency in Hz",
			},
			[]string{"channel_id", "modulation"},
		),

		upstreamSymbolRate: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hitron_upstream_symbol_rate",
				Help: "Upstream channel symbol rate",
			},
			[]string{"channel_id", "frequency", "modulation"},
		),

		systemInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hitron_system_info",
				Help: "System information",
			},
			[]string{"hardware_version", "software_version", "serial_number"},
		),
	}
}

func (c *MetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	c.downstreamPower.Describe(ch)
	c.downstreamSNR.Describe(ch)
	c.downstreamFreq.Describe(ch)
	c.downstreamCorrectables.Describe(ch)
	c.downstreamUncorrectables.Describe(ch)
	c.upstreamPower.Describe(ch)
	c.upstreamFreq.Describe(ch)
	c.upstreamSymbolRate.Describe(ch)
	c.systemInfo.Describe(ch)
}

func (c *MetricsCollector) Collect(ch chan<- prometheus.Metric) {
	// Collect downstream metrics
	dsInfo, err := c.client.GetDownstreamInfo()
	if err != nil {
		log.Printf("Failed to get downstream info: %v", err)
	} else {
		for _, channel := range dsInfo {
			labels := []string{
				strconv.Itoa(channel.ChannelID),
				fmt.Sprintf("%.0f", channel.Frequency),
				channel.Modulation,
			}

			c.downstreamPower.WithLabelValues(labels...).Set(channel.PowerLevel)
			c.downstreamSNR.WithLabelValues(labels...).Set(channel.SNR)
			c.downstreamFreq.WithLabelValues(strconv.Itoa(channel.ChannelID), channel.Modulation).Set(channel.Frequency)
			c.downstreamCorrectables.WithLabelValues(labels...).Add(float64(channel.Corrected))
			c.downstreamUncorrectables.WithLabelValues(labels...).Add(float64(channel.Uncorrectables))
		}
	}

	// Collect upstream metrics
	usInfo, err := c.client.GetUpstreamInfo()
	if err != nil {
		log.Printf("Failed to get upstream info: %v", err)
	} else {
		for _, channel := range usInfo {
			labels := []string{
				strconv.Itoa(channel.ChannelID),
				fmt.Sprintf("%.0f", channel.Frequency),
				channel.Modulation,
			}

			c.upstreamPower.WithLabelValues(labels...).Set(channel.PowerLevel)
			c.upstreamFreq.WithLabelValues(strconv.Itoa(channel.ChannelID), channel.Modulation).Set(channel.Frequency)
			c.upstreamSymbolRate.WithLabelValues(labels...).Set(channel.SymbolRate)
		}
	}

	// Collect system info
	sysInfo, err := c.client.GetSystemInfo()
	if err != nil {
		log.Printf("Failed to get system info: %v", err)
	} else {
		c.systemInfo.WithLabelValues(
			sysInfo.HardwareVersion,
			sysInfo.SoftwareVersion,
			sysInfo.SerialNumber,
		).Set(1)
	}

	// Collect all metrics
	c.downstreamPower.Collect(ch)
	c.downstreamSNR.Collect(ch)
	c.downstreamFreq.Collect(ch)
	c.downstreamCorrectables.Collect(ch)
	c.downstreamUncorrectables.Collect(ch)
	c.upstreamPower.Collect(ch)
	c.upstreamFreq.Collect(ch)
	c.upstreamSymbolRate.Collect(ch)
	c.systemInfo.Collect(ch)
}

func main() {
	flag.Parse()

	log.Printf("Starting Hitron CODA56 Prometheus Exporter")
	log.Printf("Modem host: %s", *modemHost)
	log.Printf("Listen address: %s", *listenAddr)
	log.Printf("Polling interval: %s", *interval)

	client := NewModemClient(*modemHost, *timeout)
	collector := NewMetricsCollector(client)

	prometheus.MustRegister(collector)

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
<head><title>Hitron CODA56 Exporter</title></head>
<body>
<h1>Hitron CODA56 Exporter</h1>
<p><a href="/metrics">Metrics</a></p>
</body>
</html>`))
	})

	log.Printf("Starting HTTP server on %s", *listenAddr)
	if err := http.ListenAndServe(*listenAddr, nil); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}
