# starlink_exporter

Prometheus exporter for a Starlink dish. This is a local fork of [danopstech/starlink_exporter](https://github.com/danopstech/starlink_exporter), licensed under GPL-3.0 (see `LICENSE`).

The upstream exporter ships a dish proto from 2021 and reads several values from `DishGetContext`. Current firmware rejects that call and no longer fills the old fields, so uptime, identity, and connection state came back empty. This fork loads the protobuf schema from the dish with gRPC reflection on each start, reads `get_status` and `get_history`, and publishes the fields the dish actually returns.

## Run

The dish gRPC port is `192.168.100.1:9200`. From a machine that can reach it:

```bash
go run ./cmd/starlink_exporter -address 192.168.100.1:9200 -port 9817
```

Docker:

```bash
docker build -t starlink_exporter .
docker run --rm -p 9817:9817 starlink_exporter -address 192.168.100.1:9200
```

Prometheus scrape:

```yaml
scrape_configs:
  - job_name: starlink
    scrape_interval: 15s
    scrape_timeout: 10s
    static_configs:
      - targets: ['192.168.10.112:9817']
```

`starlink_dish_up` is 1 when `get_status` succeeded. `starlink_dish_history_up` is 1 when `get_history` succeeded. `/health` reports the gRPC connection state.

Restart the exporter after a dish firmware update. Reflection runs once at startup, so a new schema is picked up on the next start.

## Metrics

Identity and link:

- `starlink_dish_info` labels: `device_id`, `hardware_version`, `software_version`, `country_code`, `utc_offset`
- `starlink_dish_uptime_seconds`, `starlink_dish_bootcount`
- `starlink_dish_state` (`0` unknown, `1` connected, `2` searching, `3` booting, `4` any other outage) and `starlink_dish_state_name{state=...}`
- `starlink_dish_outage_active`
- `starlink_dish_service` labels for class of service, disablement, NAT, software-update state, and downlink/uplink restriction
- `starlink_dish_pop_ping_latency_seconds`
- `starlink_dish_pop_ping_drop_ratio` (status field, or the newest history sample when status omits it)
- `starlink_dish_pop_ping_drop_ratio_15m` (mean of the history buffer)
- `starlink_dish_downlink_throughput_bytes` and `starlink_dish_uplink_throughput_bytes` are **bits per second**. The `_bytes` suffix is kept so older dashboards still match.
- `starlink_dish_signal_quality`, `starlink_dish_snr_above_noise_floor`, `starlink_dish_eth_speed_mbps`
- `starlink_dish_gps_valid`, `starlink_dish_gps_sats`
- boresight azimuth/elevation, desired boresight, `starlink_dish_tilt_angle_deg`, `starlink_dish_attitude_uncertainty_deg`, `starlink_dish_attitude_state`
- `starlink_dish_has_actuators`
- `starlink_dish_power_watts` from the latest history `power_in` sample

Obstruction, alerts, outages:

- `starlink_dish_currently_obstructed`, `starlink_dish_fraction_obstruction_ratio`, `starlink_dish_valid_seconds`
- `starlink_dish_prolonged_obstruction_duration_seconds`, `starlink_dish_prolonged_obstruction_interval_seconds`
- `starlink_dish_obstruction_time`, `starlink_dish_obstruction_patches_valid`
- `starlink_dish_alert{alert="..."}` for every alert flag in the dish schema, plus the original `starlink_dish_alert_*` gauges for the first six alerts
- `starlink_dish_ready{subsystem="..."}`
- `starlink_dish_history_outages` and `starlink_dish_outage_seconds{cause="..."}`

`starlink_dish_first_nonempty_slot_seconds` is the raw proto value. On current firmware that number is a unix timestamp, not a duration.

Wedge series are emitted only when the dish still sends them. SNR, cell id, POP rack, satellite id, and the other `DishGetContext` gauges are not emitted: that RPC is permission-denied, and the fields are not in `get_status`.
