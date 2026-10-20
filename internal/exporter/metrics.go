package exporter

import "github.com/prometheus/client_golang/prometheus"

const namespace = "starlink"

func dishDesc(name, help string, labels ...string) *prometheus.Desc {
	return prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "dish", name),
		help,
		labels,
		nil,
	)
}

var (
	dishUp                    = dishDesc("up", "1 if the last get_status call to the dish succeeded.")
	dishScrapeDurationSeconds = dishDesc("scrape_duration_seconds", "Time to scrape metrics from the Starlink dish.")
	dishHistoryUp             = dishDesc("history_up", "1 if the last get_history call to the dish succeeded.")

	dishInfo = dishDesc(
		"info",
		"Dish identity. Value is always 1.",
		"device_id", "hardware_version", "software_version", "country_code", "utc_offset",
	)
	dishUptimeSeconds = dishDesc("uptime_seconds", "Seconds since the dish last booted.")
	dishBootCount     = dishDesc("bootcount", "Number of times the dish has booted.")
	dishState         = dishDesc("state", "Dish connectivity as a number: 0 unknown, 1 connected, 2 searching, 3 booting, 4 other outage.")
	dishStateName     = dishDesc("state_name", "Current connectivity state. Value is 1 for the active state.", "state")
	dishService       = dishDesc(
		"service",
		"Service and policy flags reported by the dish. Value is always 1.",
		"class_of_service", "disablement_code", "nat_flag", "software_update_state", "downlink_restriction", "uplink_restriction",
	)
	dishOutageActive = dishDesc("outage_active", "1 if get_status currently includes an outage.")

	dishPopPingDropRatio      = dishDesc("pop_ping_drop_ratio", "Latest fraction of pings dropped to the point of presence. Uses get_status when the dish sets it, otherwise the newest history sample.")
	dishPopPingDropRatio15m   = dishDesc("pop_ping_drop_ratio_15m", "Mean pop ping drop fraction across the dish history buffer (about 15 minutes).")
	dishPopPingLatencySeconds = dishDesc("pop_ping_latency_seconds", "Latest pop ping latency.")
	dishUplinkThroughputBps   = dishDesc("uplink_throughput_bytes", "Latest uplink throughput in bits per second. The metric name keeps the historical _bytes suffix.")
	dishDownlinkThroughputBps = dishDesc("downlink_throughput_bytes", "Latest downlink throughput in bits per second. The metric name keeps the historical _bytes suffix.")

	dishBoreSightAzimuthDeg          = dishDesc("bore_sight_azimuth_deg", "Boresight azimuth in degrees.")
	dishBoreSightElevationDeg        = dishDesc("bore_sight_elevation_deg", "Boresight elevation in degrees.")
	dishDesiredBoreSightAzimuthDeg   = dishDesc("desired_bore_sight_azimuth_deg", "Desired boresight azimuth in degrees.")
	dishDesiredBoreSightElevationDeg = dishDesc("desired_bore_sight_elevation_deg", "Desired boresight elevation in degrees.")
	dishTiltAngleDeg                 = dishDesc("tilt_angle_deg", "Dish tilt angle in degrees.")
	dishAttitudeUncertaintyDeg       = dishDesc("attitude_uncertainty_deg", "Attitude uncertainty in degrees.")
	dishAttitudeState                = dishDesc("attitude_state", "Attitude estimation state. Value is 1 for the active state.", "state")

	dishSignalQuality      = dishDesc("signal_quality", "Dish signal_quality. 1 is the best value reported by current firmware.")
	dishSnrAboveNoiseFloor = dishDesc("snr_above_noise_floor", "1 if the dish reports SNR above the noise floor.")
	dishEthSpeedMbps       = dishDesc("eth_speed_mbps", "Ethernet link speed in Mbps.")
	dishGPSValid           = dishDesc("gps_valid", "1 if the dish reports a valid GPS fix.")
	dishGPSSats            = dishDesc("gps_sats", "Number of GPS satellites in use.")
	dishHasActuators       = dishDesc("has_actuators", "1 if the dish has motors.")
	dishFirstNonemptySlot  = dishDesc("first_nonempty_slot_seconds", "Raw seconds_to_first_nonempty_slot from the dish. Current firmware returns a unix timestamp in this field.")

	dishCurrentlyObstructed = dishDesc("currently_obstructed", "1 if the dish currently sees an obstruction.")
	dishFractionObstruction = dishDesc("fraction_obstruction_ratio", "Fraction of the obstruction map that is blocked.")
	dishValidSeconds        = dishDesc("valid_seconds", "Obstruction stats valid_s.")
	dishObstructionTime     = dishDesc("obstruction_time", "Raw obstruction_stats.time_obstructed from the dish.")
	dishObstructionPatches  = dishDesc("obstruction_patches_valid", "Obstruction stats patches_valid.")
	dishProlongedDuration   = dishDesc("prolonged_obstruction_duration_seconds", "Average duration of prolonged obstructions, in seconds.")
	dishProlongedInterval   = dishDesc("prolonged_obstruction_interval_seconds", "Average interval between prolonged obstructions, in seconds.")
	dishWedgeFraction       = dishDesc("wedge_fraction_obstruction_ratio", "Fraction obstructed in a 30 degree wedge. Absent when the dish does not send wedge data.", "wedge", "wedge_name")
	dishWedgeAbsFraction    = dishDesc("wedge_abs_fraction_obstruction_ratio", "Absolute fraction obstructed in a 30 degree wedge. Absent when the dish does not send wedge data.", "wedge", "wedge_name")

	dishSoftwareUpdateProgress = dishDesc("software_update_progress", "Software update progress from 0 to 1.")
	dishSecondsUntilReboot     = dishDesc("seconds_until_swupdate_reboot_possible", "Seconds until a software-update reboot is allowed. The dish sends -1 when no reboot is pending.")

	dishPowerWatts     = dishDesc("power_watts", "Latest power_in sample from dish history, in watts.")
	dishHistoryOutages = dishDesc("history_outages", "Number of outage records in the dish history response.")

	dishAlertMotorsStuck         = dishDesc("alert_motors_stuck", "1 if the motors-stuck alert is active.")
	dishAlertThermalThrottle     = dishDesc("alert_thermal_throttle", "1 if the thermal-throttle alert is active.")
	dishAlertThermalShutdown     = dishDesc("alert_thermal_shutdown", "1 if the thermal-shutdown alert is active.")
	dishAlertMastNotNearVertical = dishDesc("alert_mast_not_near_vertical", "1 if the mast-not-near-vertical alert is active.")
	dishAlertUnexpectedLocation  = dishDesc("alert_unexpected_location", "1 if the unexpected-location alert is active.")
	dishAlertSlowEthernet        = dishDesc("alert_slow_eth_speeds", "1 if the slow-ethernet alert is active.")
)

// legacyAlerts keeps the original metric names for the six alerts this exporter has always published.
var legacyAlerts = map[string]*prometheus.Desc{
	"motors_stuck":           dishAlertMotorsStuck,
	"thermal_throttle":       dishAlertThermalThrottle,
	"thermal_shutdown":       dishAlertThermalShutdown,
	"mast_not_near_vertical": dishAlertMastNotNearVertical,
	"unexpected_location":    dishAlertUnexpectedLocation,
	"slow_ethernet_speeds":   dishAlertSlowEthernet,
}
