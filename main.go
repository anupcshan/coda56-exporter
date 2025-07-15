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
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	modemHost  = flag.String("modem-host", "https://192.168.100.1", "Hitron CODA56 modem host URL")
	listenAddr = flag.String("listen-addr", ":2632", "Address to listen on for HTTP requests")
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

type OFDMDownstreamInfo struct {
	Receive            string `json:"receive"`
	FFTType            string `json:"ffttype"`
	Subcarr0freqFreq   string `json:"Subcarr0freqFreq"`
	PLCLock            string `json:"plclock"`
	NCPLock            string `json:"ncplock"`
	MDC1Lock           string `json:"mdc1lock"`
	PLCPower           string `json:"plcpower"`
	SNR                string `json:"SNR"`
	DSoctets           string `json:"dsoctets"`
	Correcteds         string `json:"correcteds"`
	Uncorrect          string `json:"uncorrect"`
}

type OFDMUpstreamInfo struct {
	USCHIndex    string `json:"uschindex"`
	State        string `json:"state"`
	Frequency    string `json:"frequency"`
	DigAtten     string `json:"digAtten"`
	DigAttenBo   string `json:"digAttenBo"`
	ChannelBw    string `json:"channelBw"`
	RepPower     string `json:"repPower"`
	RepPower1_6  string `json:"repPower1_6"`
	FFTVal       string `json:"fftVal"`
}

type LinkStatus struct {
	LinkStatus string `json:"LinkStatus"`
	LinkDuplex string `json:"LinkDuplex"`
	LinkSpeed  string `json:"LinkSpeed"`
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

func (m *ModemClient) parseOFDMDownstreamInfo(data []byte) ([]OFDMDownstreamInfo, error) {
	var channels []OFDMDownstreamInfo
	if err := json.Unmarshal(data, &channels); err != nil {
		return nil, fmt.Errorf("failed to parse OFDM downstream info JSON: %w", err)
	}
	log.Printf("Parsed %d OFDM downstream channels", len(channels))
	return channels, nil
}

func (m *ModemClient) parseOFDMUpstreamInfo(data []byte) ([]OFDMUpstreamInfo, error) {
	var channels []OFDMUpstreamInfo
	if err := json.Unmarshal(data, &channels); err != nil {
		return nil, fmt.Errorf("failed to parse OFDM upstream info JSON: %w", err)
	}
	log.Printf("Parsed %d OFDM upstream channels", len(channels))
	return channels, nil
}

func (m *ModemClient) parseLinkStatus(data []byte) (*LinkStatus, error) {
	var linkStatusArray []LinkStatus
	if err := json.Unmarshal(data, &linkStatusArray); err != nil {
		return nil, fmt.Errorf("failed to parse link status JSON: %w", err)
	}
	if len(linkStatusArray) == 0 {
		return nil, fmt.Errorf("empty link status response")
	}
	log.Println("Parsed link status")
	return &linkStatusArray[0], nil
}

func (m *ModemClient) GetOFDMDownstreamInfo() ([]OFDMDownstreamInfo, error) {
	data, err := m.get("dsofdminfo.asp")
	if err != nil {
		return nil, err
	}
	return m.parseOFDMDownstreamInfo(data)
}

func (m *ModemClient) GetOFDMUpstreamInfo() ([]OFDMUpstreamInfo, error) {
	data, err := m.get("usofdminfo.asp")
	if err != nil {
		return nil, err
	}
	return m.parseOFDMUpstreamInfo(data)
}

func (m *ModemClient) GetLinkStatus() (*LinkStatus, error) {
	data, err := m.get("getLinkStatus.asp")
	if err != nil {
		return nil, err
	}
	return m.parseLinkStatus(data)
}

// parseComplexOctets parses QAM downstream octet format like "53 * 2e32 + 4142950845"
func parseComplexOctets(octetsStr string) int64 {
	// Handle simple numeric format first
	if simple, err := strconv.ParseInt(octetsStr, 10, 64); err == nil {
		return simple
	}
	
	// Parse complex format: "53 * 2e32 + 4142950845"
	// Split on " + " to get the two parts
	parts := strings.Split(octetsStr, " + ")
	if len(parts) != 2 {
		return 0
	}
	
	// Parse the high part: "53 * 2e32"
	highParts := strings.Split(parts[0], " * ")
	if len(highParts) != 2 {
		return 0
	}
	
	multiplier, err1 := strconv.ParseFloat(highParts[0], 64)
	factor, err2 := strconv.ParseFloat(highParts[1], 64)
	lowPart, err3 := strconv.ParseFloat(parts[1], 64)
	
	if err1 != nil || err2 != nil || err3 != nil {
		return 0
	}
	
	// Calculate: multiplier * factor + lowPart
	// Use float64 for calculation to handle large numbers, then convert
	result := multiplier*factor + lowPart
	
	// For very large numbers, just return the low part since the high part 
	// represents data transferred over a very long time and may overflow
	if result > 9.223372036854775e+18 { // Close to int64 max
		return int64(lowPart)
	}
	
	return int64(result)
}

type MetricsCollector struct {
	client *ModemClient

	// Downstream metrics
	downstreamPower          *prometheus.GaugeVec
	downstreamSNR            *prometheus.GaugeVec
	downstreamFreq           *prometheus.GaugeVec
	downstreamCorrectables   *prometheus.CounterVec
	downstreamUncorrectables *prometheus.CounterVec
	downstreamOctets         *prometheus.GaugeVec

	// Upstream metrics
	upstreamPower      *prometheus.GaugeVec
	upstreamFreq       *prometheus.GaugeVec
	upstreamSymbolRate *prometheus.GaugeVec

	// OFDM Downstream metrics
	ofdmDownstreamPower          *prometheus.GaugeVec
	ofdmDownstreamSNR            *prometheus.GaugeVec
	ofdmDownstreamFreq           *prometheus.GaugeVec
	ofdmDownstreamCorrectables   *prometheus.CounterVec
	ofdmDownstreamUncorrectables *prometheus.CounterVec
	ofdmDownstreamOctets         *prometheus.GaugeVec
	ofdmDownstreamLocks          *prometheus.GaugeVec

	// OFDM Upstream metrics
	ofdmUpstreamPower     *prometheus.GaugeVec
	ofdmUpstreamFreq      *prometheus.GaugeVec
	ofdmUpstreamBandwidth *prometheus.GaugeVec
	ofdmUpstreamState     *prometheus.GaugeVec

	// Link status metrics
	linkStatus *prometheus.GaugeVec
	linkSpeed  *prometheus.GaugeVec

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

		downstreamOctets: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hitron_downstream_octets_bytes",
				Help: "Number of octets (bytes) received on downstream channel",
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

		// OFDM Downstream metrics
		ofdmDownstreamPower: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hitron_ofdm_downstream_power_dbmv",
				Help: "OFDM downstream channel power level in dBmV",
			},
			[]string{"receive", "frequency", "fft_type"},
		),

		ofdmDownstreamSNR: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hitron_ofdm_downstream_snr_db",
				Help: "OFDM downstream channel signal-to-noise ratio in dB",
			},
			[]string{"receive", "frequency", "fft_type"},
		),

		ofdmDownstreamFreq: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hitron_ofdm_downstream_frequency_hz",
				Help: "OFDM downstream channel frequency in Hz",
			},
			[]string{"receive", "fft_type"},
		),

		ofdmDownstreamCorrectables: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "hitron_ofdm_downstream_correctables_total",
				Help: "Total number of correctable errors on OFDM downstream channel",
			},
			[]string{"receive", "frequency", "fft_type"},
		),

		ofdmDownstreamUncorrectables: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "hitron_ofdm_downstream_uncorrectables_total",
				Help: "Total number of uncorrectable errors on OFDM downstream channel",
			},
			[]string{"receive", "frequency", "fft_type"},
		),

		ofdmDownstreamOctets: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hitron_ofdm_downstream_octets_bytes",
				Help: "Number of octets (bytes) received on OFDM downstream channel",
			},
			[]string{"receive", "frequency", "fft_type"},
		),

		ofdmDownstreamLocks: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hitron_ofdm_downstream_locks",
				Help: "OFDM downstream channel lock status (1 = locked, 0 = unlocked)",
			},
			[]string{"receive", "frequency", "lock_type"},
		),

		// OFDM Upstream metrics
		ofdmUpstreamPower: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hitron_ofdm_upstream_power_dbmv",
				Help: "OFDM upstream channel power level in dBmV",
			},
			[]string{"usch_index", "frequency", "state"},
		),

		ofdmUpstreamFreq: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hitron_ofdm_upstream_frequency_hz",
				Help: "OFDM upstream channel frequency in Hz",
			},
			[]string{"usch_index", "state"},
		),

		ofdmUpstreamBandwidth: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hitron_ofdm_upstream_bandwidth_mhz",
				Help: "OFDM upstream channel bandwidth in MHz",
			},
			[]string{"usch_index", "frequency", "state"},
		),

		ofdmUpstreamState: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hitron_ofdm_upstream_state",
				Help: "OFDM upstream channel state (1 = operate, 0 = disabled)",
			},
			[]string{"usch_index", "frequency"},
		),

		// Link status metrics
		linkStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hitron_link_status",
				Help: "Link status (1 = up, 0 = down)",
			},
			[]string{"duplex"},
		),

		linkSpeed: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "hitron_link_speed_mbps",
				Help: "Link speed in Mbps",
			},
			[]string{"duplex"},
		),
	}
}

func (c *MetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	c.downstreamPower.Describe(ch)
	c.downstreamSNR.Describe(ch)
	c.downstreamFreq.Describe(ch)
	c.downstreamCorrectables.Describe(ch)
	c.downstreamUncorrectables.Describe(ch)
	c.downstreamOctets.Describe(ch)
	c.upstreamPower.Describe(ch)
	c.upstreamFreq.Describe(ch)
	c.upstreamSymbolRate.Describe(ch)
	c.ofdmDownstreamPower.Describe(ch)
	c.ofdmDownstreamSNR.Describe(ch)
	c.ofdmDownstreamFreq.Describe(ch)
	c.ofdmDownstreamCorrectables.Describe(ch)
	c.ofdmDownstreamUncorrectables.Describe(ch)
	c.ofdmDownstreamOctets.Describe(ch)
	c.ofdmDownstreamLocks.Describe(ch)
	c.ofdmUpstreamPower.Describe(ch)
	c.ofdmUpstreamFreq.Describe(ch)
	c.ofdmUpstreamBandwidth.Describe(ch)
	c.ofdmUpstreamState.Describe(ch)
	c.linkStatus.Describe(ch)
	c.linkSpeed.Describe(ch)
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
			
			// Parse complex octet format: "53 * 2e32 + 4142950845"
			octets := parseComplexOctets(channel.DSoctets)
			
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
			c.downstreamOctets.WithLabelValues(labels...).Set(float64(octets))
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

	// Collect OFDM downstream metrics
	ofdmDsInfo, err := c.client.GetOFDMDownstreamInfo()
	if err != nil {
		log.Printf("Failed to get OFDM downstream info: %v", err)
	} else {
		for _, channel := range ofdmDsInfo {
			// Parse numeric values from strings
			frequency, _ := strconv.ParseFloat(strings.TrimSpace(channel.Subcarr0freqFreq), 64)
			powerLevel, _ := strconv.ParseFloat(channel.PLCPower, 64)
			snr, _ := strconv.ParseFloat(channel.SNR, 64)
			corrected, _ := strconv.ParseInt(channel.Correcteds, 10, 64)
			uncorrect, _ := strconv.ParseInt(channel.Uncorrect, 10, 64)
			
			// Parse simple octet format for OFDM: "53196813856"
			octets, _ := strconv.ParseInt(channel.DSoctets, 10, 64)

			labels := []string{
				channel.Receive,
				strings.TrimSpace(channel.Subcarr0freqFreq),
				channel.FFTType,
			}

			c.ofdmDownstreamPower.WithLabelValues(labels...).Set(powerLevel)
			c.ofdmDownstreamSNR.WithLabelValues(labels...).Set(snr)
			c.ofdmDownstreamFreq.WithLabelValues(channel.Receive, channel.FFTType).Set(frequency)
			c.ofdmDownstreamCorrectables.WithLabelValues(labels...).Add(float64(corrected))
			c.ofdmDownstreamUncorrectables.WithLabelValues(labels...).Add(float64(uncorrect))
			c.ofdmDownstreamOctets.WithLabelValues(labels...).Set(float64(octets))

			// Lock status metrics
			lockLabels := []string{channel.Receive, strings.TrimSpace(channel.Subcarr0freqFreq)}
			plcLock := 0.0
			if strings.TrimSpace(channel.PLCLock) == "YES" {
				plcLock = 1.0
			}
			ncpLock := 0.0
			if strings.TrimSpace(channel.NCPLock) == "YES" {
				ncpLock = 1.0
			}
			mdc1Lock := 0.0
			if strings.TrimSpace(channel.MDC1Lock) == "YES" {
				mdc1Lock = 1.0
			}

			c.ofdmDownstreamLocks.WithLabelValues(append(lockLabels, "plc")...).Set(plcLock)
			c.ofdmDownstreamLocks.WithLabelValues(append(lockLabels, "ncp")...).Set(ncpLock)
			c.ofdmDownstreamLocks.WithLabelValues(append(lockLabels, "mdc1")...).Set(mdc1Lock)
		}
	}

	// Collect OFDM upstream metrics
	ofdmUsInfo, err := c.client.GetOFDMUpstreamInfo()
	if err != nil {
		log.Printf("Failed to get OFDM upstream info: %v", err)
	} else {
		for _, channel := range ofdmUsInfo {
			// Parse numeric values from strings
			frequency, _ := strconv.ParseFloat(channel.Frequency, 64)
			repPower, _ := strconv.ParseFloat(strings.TrimSpace(channel.RepPower), 64)
			bandwidth, _ := strconv.ParseFloat(strings.TrimSpace(channel.ChannelBw), 64)

			state := strings.TrimSpace(channel.State)
			stateValue := 0.0
			if state == "OPERATE" {
				stateValue = 1.0
			}

			labels := []string{
				channel.USCHIndex,
				channel.Frequency,
				state,
			}

			if frequency > 0 { // Only collect metrics for active channels
				c.ofdmUpstreamPower.WithLabelValues(labels...).Set(repPower)
				c.ofdmUpstreamFreq.WithLabelValues(channel.USCHIndex, state).Set(frequency)
				c.ofdmUpstreamBandwidth.WithLabelValues(labels...).Set(bandwidth)
			}
			c.ofdmUpstreamState.WithLabelValues(channel.USCHIndex, channel.Frequency).Set(stateValue)
		}
	}

	// Collect link status
	linkInfo, err := c.client.GetLinkStatus()
	if err != nil {
		log.Printf("Failed to get link status: %v", err)
	} else {
		// Parse link status
		status := 0.0
		if linkInfo.LinkStatus == "Up" {
			status = 1.0
		}

		// Parse link speed (extract number from "2500Mbps")
		speedStr := strings.TrimSuffix(linkInfo.LinkSpeed, "Mbps")
		speed, _ := strconv.ParseFloat(speedStr, 64)

		duplex := linkInfo.LinkDuplex
		c.linkStatus.WithLabelValues(duplex).Set(status)
		c.linkSpeed.WithLabelValues(duplex).Set(speed)
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
	c.downstreamOctets.Collect(ch)
	c.upstreamPower.Collect(ch)
	c.upstreamFreq.Collect(ch)
	c.upstreamSymbolRate.Collect(ch)
	c.ofdmDownstreamPower.Collect(ch)
	c.ofdmDownstreamSNR.Collect(ch)
	c.ofdmDownstreamFreq.Collect(ch)
	c.ofdmDownstreamCorrectables.Collect(ch)
	c.ofdmDownstreamUncorrectables.Collect(ch)
	c.ofdmDownstreamOctets.Collect(ch)
	c.ofdmDownstreamLocks.Collect(ch)
	c.ofdmUpstreamPower.Collect(ch)
	c.ofdmUpstreamFreq.Collect(ch)
	c.ofdmUpstreamBandwidth.Collect(ch)
	c.ofdmUpstreamState.Collect(ch)
	c.linkStatus.Collect(ch)
	c.linkSpeed.Collect(ch)
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
