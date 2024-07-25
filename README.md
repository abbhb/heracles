# Heracles

Prometheus exporter integration testing tools.

## Getting started

Heracles automatically launches the relevant services based on the docker compose configuration, automatically collects metrics from the exporter service, and performs validation according to the configuration file requirements.

[![asciicast](https://asciinema.org/a/DrMVWSmRcIxMj0TUnKRDChWr5.svg)](https://asciinema.org/a/DrMVWSmRcIxMj0TUnKRDChWr5)

## Configuration

See [config-example.yml](config-example.yml)

## Report

Example report(default: `heracles-report.yml`):

```yaml
success: true
metrics:
  pg_up:
    name: pg_up
    help: Whether the last scrape of metrics from PostgreSQL was able to connect to the server (1 for yes, 0 for no).
    type: 1
    metric:
      - label: []
        gauge:
          value: 1
results:
  DisallowEmptyMetricsChecker: ok!
```