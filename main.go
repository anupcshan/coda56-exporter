package main

import (
	"crypto/tls"
	"encoding/json"
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
	PortID         string `json:"portId"`
	Frequency      string `json:"frequency"`
	Modulation     string `json:"modulation"`
	SignalStrength string `json:"signalStrength"`
	SNR            string `json:"snr"`
	DSoctets       string `json:"dsoctets"`
	Correcteds     string `json:"correcteds"`
	Uncorrect      string `json:"uncorrect"`
	ChannelID      string `json:"channelId"`
}

type UpstreamInfo struct {
	PortID         string `json:"portId"`
	Frequency      string `json:"frequency"`
	Bandwidth      string `json:"bandwidth"`
	ModType        string `json:"modtype"`
	ScdmaMode      string `json:"scdmaMode"`
	SignalStrength string `json:"signalStrength"`
	ChannelID      string `json:"channelId"`
}

type SystemInfo struct {
	HWVersion     string `json:"hwVersion"`
	SWVersion     string `json:"swVersion"`
	SerialNumber  string `json:"serialNumber"`
	RFMac         string `json:"rfMac"`
	WanIP         string `json:"wanIp"`
	SystemUptime  string `json:"systemUptime"`
	SystemTime    string `json:"systemTime"`
	Timezone      string `json:"timezone"`
	WRecPkt       string `json:"WRecPkt"`
	WSendPkt      string `json:"WSendPkt"`
	LanIP         string `json:"lanIp"`
	LRecPkt       string `json:"LRecPkt"`
	LSendPkt      string `json:"LSendPkt"`
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
	var channels []DownstreamInfo
	if err := json.Unmarshal(data, &channels); err != nil {
		return nil, fmt.Errorf("failed to parse downstream info JSON: %w", err)
	}
	log.Printf("Parsed %d downstream channels", len(channels))
	return channels, nil
}

func (m *ModemClient) parseUpstreamInfo(data []byte) ([]UpstreamInfo, error) {
	var channels []UpstreamInfo
	if err := json.Unmarshal(data, &channels); err != nil {
		return nil, fmt.Errorf("failed to parse upstream info JSON: %w", err)
	}
	log.Printf("Parsed %d upstream channels", len(channels))
	return channels, nil
}

func (m *ModemClient) parseSystemInfo(data []byte) (*SystemInfo, error) {
	var sysInfoArray []SystemInfo
	if err := json.Unmarshal(data, &sysInfoArray); err != nil {
		return nil, fmt.Errorf("failed to parse system info JSON: %w", err)
	}
	if len(sysInfoArray) == 0 {
		return nil, fmt.Errorf("empty system info response")
	}
	log.Println("Parsed system info")
	return &sysInfoArray[0], nil
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
			// Parse numeric values from strings
			frequency, _ := strconv.ParseFloat(channel.Frequency, 64)
			powerLevel, _ := strconv.ParseFloat(channel.SignalStrength, 64)
			snr, _ := strconv.ParseFloat(channel.SNR, 64)
			corrected, _ := strconv.ParseInt(channel.Correcteds, 10, 64)
			uncorrect, _ := strconv.ParseInt(channel.Uncorrect, 10, 64)
			
			labels := []string{
				channel.ChannelID,
				channel.Frequency,
				channel.Modulation,
			}

			c.downstreamPower.WithLabelValues(labels...).Set(powerLevel)
			c.downstreamSNR.WithLabelValues(labels...).Set(snr)
			c.downstreamFreq.WithLabelValues(channel.ChannelID, channel.Modulation).Set(frequency)
			c.downstreamCorrectables.WithLabelValues(labels...).Add(float64(corrected))
			c.downstreamUncorrectables.WithLabelValues(labels...).Add(float64(uncorrect))
		}
	}

	// Collect upstream metrics
	usInfo, err := c.client.GetUpstreamInfo()
	if err != nil {
		log.Printf("Failed to get upstream info: %v", err)
	} else {
		for _, channel := range usInfo {
			// Parse numeric values from strings
			frequency, _ := strconv.ParseFloat(channel.Frequency, 64)
			powerLevel, _ := strconv.ParseFloat(channel.SignalStrength, 64)
			bandwidth, _ := strconv.ParseFloat(channel.Bandwidth, 64)
			
			labels := []string{
				channel.ChannelID,
				channel.Frequency,
				channel.ModType,
			}

			c.upstreamPower.WithLabelValues(labels...).Set(powerLevel)
			c.upstreamFreq.WithLabelValues(channel.ChannelID, channel.ModType).Set(frequency)
			c.upstreamSymbolRate.WithLabelValues(labels...).Set(bandwidth)
		}
	}

	// Collect system info
	sysInfo, err := c.client.GetSystemInfo()
	if err != nil {
		log.Printf("Failed to get system info: %v", err)
	} else {
		c.systemInfo.WithLabelValues(
			sysInfo.HWVersion,
			sysInfo.SWVersion,
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
