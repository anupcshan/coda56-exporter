# Hitron CODA56 Prometheus Exporter

A Prometheus exporter for the Hitron CODA56 cable modem that collects metrics from the modem's HTTP API and exposes them in Prometheus format.

## Features

- Collects downstream channel metrics (power, SNR, frequency, error counts)
- Collects upstream channel metrics (power, frequency, symbol rate)
- Collects system information (hardware/software versions, serial number)
- Configurable polling intervals
- TLS support for HTTPS connections to the modem

## Usage

```bash
# Build the exporter
go build -o coda56-exporter

# Run with default settings (modem at https://192.168.100.1)
./coda56-exporter

# Run with custom settings
./coda56-exporter \
  -modem-host https://192.168.100.1 \
  -listen-addr :8080 \
  -interval 30s \
  -timeout 10s
```

## Command Line Options

- `-modem-host`: Hitron CODA56 modem host URL (default: https://192.168.100.1)
- `-listen-addr`: Address to listen on for HTTP requests (default: :8080)
- `-interval`: Polling interval (default: 30s)
- `-timeout`: HTTP request timeout (default: 10s)

## Metrics

The exporter exposes the following metrics:

### QAM Downstream Channel Metrics (32 channels)
- `hitron_downstream_power_dbmv`: Power level in dBmV
- `hitron_downstream_snr_db`: Signal-to-noise ratio in dB
- `hitron_downstream_frequency_hz`: Frequency in Hz
- `hitron_downstream_correctables_total`: Total correctable errors
- `hitron_downstream_uncorrectables_total`: Total uncorrectable errors

### QAM Upstream Channel Metrics (4 channels)
- `hitron_upstream_power_dbmv`: Power level in dBmV
- `hitron_upstream_frequency_hz`: Frequency in Hz
- `hitron_upstream_symbol_rate`: Symbol rate (bandwidth)

### OFDM Downstream Channel Metrics (2 channels)
- `hitron_ofdm_downstream_power_dbmv`: Power level in dBmV
- `hitron_ofdm_downstream_snr_db`: Signal-to-noise ratio in dB
- `hitron_ofdm_downstream_frequency_hz`: Frequency in Hz
- `hitron_ofdm_downstream_correctables_total`: Total correctable errors
- `hitron_ofdm_downstream_uncorrectables_total`: Total uncorrectable errors
- `hitron_ofdm_downstream_locks`: Lock status for PLC/NCP/MDC1 (1=locked, 0=unlocked)

### OFDM Upstream Channel Metrics (2 channels)
- `hitron_ofdm_upstream_power_dbmv`: Power level in dBmV
- `hitron_ofdm_upstream_frequency_hz`: Frequency in Hz
- `hitron_ofdm_upstream_bandwidth_mhz`: Channel bandwidth in MHz
- `hitron_ofdm_upstream_state`: Channel state (1=operate, 0=disabled)

### Link Status Metrics
- `hitron_link_status`: Link status (1=up, 0=down)
- `hitron_link_speed_mbps`: Link speed in Mbps

### System Metrics
- `hitron_system_info`: System information with labels for hardware/software versions

## API Endpoints

The exporter polls the following modem API endpoints:

- `/data/dsinfo.asp`: Downstream QAM channel information (32 channels)
- `/data/dsofdminfo.asp`: Downstream OFDM channel details (2 channels)
- `/data/usinfo.asp`: Upstream QAM channel information (4 channels)
- `/data/usofdminfo.asp`: Upstream OFDM channel details (2 channels)
- `/data/getSysInfo.asp`: System information and hardware details
- `/data/getLinkStatus.asp`: Link connection status and speed

## Network Requirements

The Hitron CODA56 modem requires requests to come from the 192.168.100.x network. If your monitoring system is on a different network, you may need to configure routing or use a proxy.

## Development Notes

The current implementation includes placeholder parsers for the HTML/JavaScript responses from the modem's API endpoints. The actual parsing logic needs to be implemented based on the specific format returned by your modem firmware version.

To implement the parsers:

1. Capture sample responses from each endpoint
2. Analyze the HTML/JavaScript structure
3. Implement appropriate parsing logic in the `parse*Info` methods

## License

MIT License