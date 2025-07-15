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

### Downstream Channel Metrics
- `hitron_downstream_power_dbmv`: Power level in dBmV
- `hitron_downstream_snr_db`: Signal-to-noise ratio in dB
- `hitron_downstream_frequency_hz`: Frequency in Hz
- `hitron_downstream_correctables_total`: Total correctable errors
- `hitron_downstream_uncorrectables_total`: Total uncorrectable errors

### Upstream Channel Metrics
- `hitron_upstream_power_dbmv`: Power level in dBmV
- `hitron_upstream_frequency_hz`: Frequency in Hz
- `hitron_upstream_symbol_rate`: Symbol rate

### System Metrics
- `hitron_system_info`: System information with labels for hardware/software versions

## API Endpoints

The exporter polls the following modem API endpoints:

- `/dsinfo.asp`: Downstream QAM channel information
- `/dsofdminfo.asp`: Downstream OFDM channel details
- `/usinfo.asp`: Upstream QAM channel information
- `/usofdminfo.asp`: Upstream OFDM channel details
- `/getSysInfo.asp`: System information
- `/getLinkStatus.asp`: Link speed status

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