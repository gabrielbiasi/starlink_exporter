package exporter

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/dynamic/grpcdynamic"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	// DishAddress is the dish gRPC endpoint.
	DishAddress = "192.168.100.1:9200"

	deviceService = "SpaceX.API.Device.Device"
	handleMethod  = "Handle"
)

// Exporter collects Starlink stats from the dish and exports them as Prometheus metrics.
// The protocol is loaded from the dish with gRPC reflection, so field additions in newer
// firmware stay readable without regenerating a vendored proto.
type Exporter struct {
	Conn     *grpc.ClientConn
	DishID   string
	Hardware string
	Software string

	stub   grpcdynamic.Stub
	method *desc.MethodDescriptor

	mu            sync.Mutex
	alerts        *prometheus.GaugeVec
	ready         *prometheus.GaugeVec
	outageSeconds *prometheus.GaugeVec
}

// New dials the dish, loads its protobuf schema, and reads identity from get_status.
func New(address string) (*Exporter, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(32<<20)),
	)
	if err != nil {
		return nil, fmt.Errorf("dial dish: %w", err)
	}

	e := &Exporter{
		Conn: conn,
		stub: grpcdynamic.NewStub(conn),
		alerts: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "dish",
			Name:      "alert",
			Help:      "Dish alert flag. 1 means the alert is active.",
		}, []string{"alert"}),
		ready: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "dish",
			Name:      "ready",
			Help:      "Dish subsystem ready flag. 1 means ready.",
		}, []string{"subsystem"}),
		outageSeconds: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "dish",
			Name:      "outage_seconds",
			Help:      "Sum of outage durations in the dish history response, by cause.",
		}, []string{"cause"}),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.resolve(ctx); err != nil {
		conn.Close()
		return nil, err
	}

	status, err := e.call(ctx, "get_status")
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("read dish status: %w", err)
	}
	info := child(status, "device_info")
	e.DishID = stringField(info, "id")
	e.Hardware = stringField(info, "hardware_version")
	e.Software = stringField(info, "software_version")
	return e, nil
}

func (e *Exporter) resolve(ctx context.Context) error {
	refClient := grpcreflect.NewClientAuto(ctx, e.Conn)
	defer refClient.Reset()

	svc, err := refClient.ResolveService(deviceService)
	if err != nil {
		return fmt.Errorf("reflect dish service: %w", err)
	}
	method := svc.FindMethodByName(handleMethod)
	if method == nil {
		return fmt.Errorf("dish service has no %s method", handleMethod)
	}
	e.method = method
	return nil
}

func (e *Exporter) call(ctx context.Context, requestField string) (*dynamic.Message, error) {
	in := e.method.GetInputType()
	reqField := in.FindFieldByName(requestField)
	if reqField == nil || reqField.GetMessageType() == nil {
		return nil, fmt.Errorf("dish request has no %s message", requestField)
	}
	req := dynamic.NewMessage(in)
	req.SetField(reqField, dynamic.NewMessage(reqField.GetMessageType()))

	resp, err := e.stub.InvokeRpc(ctx, e.method, req)
	if err != nil {
		return nil, err
	}
	dm, ok := resp.(*dynamic.Message)
	if !ok {
		return nil, fmt.Errorf("unexpected %s response type %T", requestField, resp)
	}
	inner := firstSubmessage(dm)
	if inner == nil {
		return nil, fmt.Errorf("%s response did not include a message", requestField)
	}
	return inner, nil
}

func firstSubmessage(m *dynamic.Message) *dynamic.Message {
	for _, fd := range m.GetMessageDescriptor().GetFields() {
		if fd.IsRepeated() || fd.GetMessageType() == nil || !m.HasField(fd) {
			continue
		}
		v, err := m.TryGetField(fd)
		if err != nil {
			continue
		}
		if dm, ok := v.(*dynamic.Message); ok {
			return dm
		}
	}
	return nil
}

// Describe implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range []*prometheus.Desc{
		dishUp,
		dishScrapeDurationSeconds,
		dishHistoryUp,
		dishInfo,
		dishUptimeSeconds,
		dishBootCount,
		dishState,
		dishStateName,
		dishService,
		dishOutageActive,
		dishPopPingDropRatio,
		dishPopPingDropRatio15m,
		dishPopPingLatencySeconds,
		dishUplinkThroughputBps,
		dishDownlinkThroughputBps,
		dishBoreSightAzimuthDeg,
		dishBoreSightElevationDeg,
		dishDesiredBoreSightAzimuthDeg,
		dishDesiredBoreSightElevationDeg,
		dishTiltAngleDeg,
		dishAttitudeUncertaintyDeg,
		dishAttitudeState,
		dishSignalQuality,
		dishSnrAboveNoiseFloor,
		dishEthSpeedMbps,
		dishGPSValid,
		dishGPSSats,
		dishHasActuators,
		dishFirstNonemptySlot,
		dishCurrentlyObstructed,
		dishFractionObstruction,
		dishValidSeconds,
		dishObstructionTime,
		dishObstructionPatches,
		dishProlongedDuration,
		dishProlongedInterval,
		dishWedgeFraction,
		dishWedgeAbsFraction,
		dishSoftwareUpdateProgress,
		dishSecondsUntilReboot,
		dishPowerWatts,
		dishHistoryOutages,
		dishAlertMotorsStuck,
		dishAlertThermalThrottle,
		dishAlertThermalShutdown,
		dishAlertMastNotNearVertical,
		dishAlertUnexpectedLocation,
		dishAlertSlowEthernet,
	} {
		ch <- desc
	}
	e.alerts.Describe(ch)
	e.ready.Describe(ch)
	e.outageSeconds.Describe(ch)
}

// Collect implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mu.Lock()
	defer e.mu.Unlock()

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status, err := e.call(ctx, "get_status")
	if err != nil {
		log.Errorf("get_status failed: %s", err)
		emit(ch, dishUp, 0)
		return
	}

	history, histErr := e.call(ctx, "get_history")
	if histErr != nil {
		log.Warnf("get_history failed: %s", histErr)
	}

	e.emitStatus(ch, status)
	if histErr == nil {
		e.emitHistory(ch, history, status)
		emit(ch, dishHistoryUp, 1)
	} else {
		e.emitDropRatio(ch, status, nil)
		emit(ch, dishHistoryUp, 0)
	}

	emit(ch, dishUp, 1)
	emit(ch, dishScrapeDurationSeconds, time.Since(start).Seconds())
}

func (e *Exporter) emitStatus(ch chan<- prometheus.Metric, status *dynamic.Message) {
	info := child(status, "device_info")
	utc := ""
	if offset, ok := floatField(info, "utc_offset_s"); ok {
		utc = strconv.Itoa(int(offset))
	}
	ch <- prometheus.MustNewConstMetric(
		dishInfo, prometheus.GaugeValue, 1,
		orUnknown(stringField(info, "id")),
		orUnknown(stringField(info, "hardware_version")),
		orUnknown(stringField(info, "software_version")),
		orUnknown(stringField(info, "country_code")),
		orUnknown(utc),
	)
	emitFloat(ch, dishUptimeSeconds, child(status, "device_state"), "uptime_s")
	emitFloat(ch, dishBootCount, info, "bootcount")

	stateName := "CONNECTED"
	outageActive := 0.0
	if outage := child(status, "outage"); outage != nil {
		outageActive = 1
		if cause := enumName(outage, "cause"); cause != "" {
			stateName = cause
		} else {
			stateName = "UNKNOWN"
		}
	}
	emit(ch, dishState, stateNumber(stateName))
	ch <- prometheus.MustNewConstMetric(dishStateName, prometheus.GaugeValue, 1, stateName)
	emit(ch, dishOutageActive, outageActive)

	ch <- prometheus.MustNewConstMetric(
		dishService, prometheus.GaugeValue, 1,
		orUnknown(enumName(status, "class_of_service")),
		orUnknown(enumName(status, "disablement_code")),
		orUnknown(enumName(status, "nat_flag")),
		orUnknown(enumName(status, "software_update_state")),
		orUnknown(enumName(status, "dl_bandwidth_restricted_reason")),
		orUnknown(enumName(status, "ul_bandwidth_restricted_reason")),
	)

	if ms, ok := floatField(status, "pop_ping_latency_ms"); ok {
		emit(ch, dishPopPingLatencySeconds, ms/1000)
	}
	emitFloat(ch, dishUplinkThroughputBps, status, "uplink_throughput_bps")
	emitFloat(ch, dishDownlinkThroughputBps, status, "downlink_throughput_bps")
	emitFloat(ch, dishBoreSightAzimuthDeg, status, "boresight_azimuth_deg")
	emitFloat(ch, dishBoreSightElevationDeg, status, "boresight_elevation_deg")
	emitFloat(ch, dishSignalQuality, status, "signal_quality")
	emitBool(ch, dishSnrAboveNoiseFloor, status, "is_snr_above_noise_floor")
	emitFloat(ch, dishEthSpeedMbps, status, "eth_speed_mbps")
	emitFloat(ch, dishFirstNonemptySlot, status, "seconds_to_first_nonempty_slot")
	emitActuators(ch, status)

	if gps := child(status, "gps_stats"); gps != nil {
		valid, _ := boolField(gps, "gps_valid")
		emit(ch, dishGPSValid, gaugeBool(valid))
		emitFloat(ch, dishGPSSats, gps, "gps_sats")
	}

	if align := child(status, "alignment_stats"); align != nil {
		emitFloat(ch, dishDesiredBoreSightAzimuthDeg, align, "desired_boresight_azimuth_deg")
		emitFloat(ch, dishDesiredBoreSightElevationDeg, align, "desired_boresight_elevation_deg")
		emitFloat(ch, dishTiltAngleDeg, align, "tilt_angle_deg")
		emitFloat(ch, dishAttitudeUncertaintyDeg, align, "attitude_uncertainty_deg")
		if state := enumName(align, "attitude_estimation_state"); state != "" {
			ch <- prometheus.MustNewConstMetric(dishAttitudeState, prometheus.GaugeValue, 1, state)
		}
	}

	if update := child(status, "software_update_stats"); update != nil {
		emitFloat(ch, dishSoftwareUpdateProgress, update, "software_update_progress")
	}
	emitFloat(ch, dishSecondsUntilReboot, status, "seconds_until_swupdate_reboot_possible")

	e.emitObstruction(ch, child(status, "obstruction_stats"))
	e.emitAlerts(ch, child(status, "alerts"))
	e.emitReady(ch, child(status, "ready_states"))
}

func (e *Exporter) emitHistory(ch chan<- prometheus.Metric, history, status *dynamic.Message) {
	drops, err := floatList(history, "pop_ping_drop_rate")
	if err != nil {
		log.Warnf("history pop_ping_drop_rate: %s", err)
	}
	e.emitDropRatio(ch, status, drops)
	if avg, ok := mean(drops); ok {
		emit(ch, dishPopPingDropRatio15m, avg)
	}

	power, err := floatList(history, "power_in")
	if err != nil {
		log.Warnf("history power_in: %s", err)
	} else if watts, ok := last(power); ok {
		emit(ch, dishPowerWatts, watts)
	}

	outages, err := messageList(history, "outages")
	if err != nil {
		log.Warnf("history outages: %s", err)
		return
	}
	emit(ch, dishHistoryOutages, float64(len(outages)))

	totals := map[string]float64{}
	for _, outage := range outages {
		cause := enumName(outage, "cause")
		if cause == "" {
			cause = "UNKNOWN"
		}
		ns, ok := floatField(outage, "duration_ns")
		if !ok {
			continue
		}
		totals[cause] += ns / 1e9
	}
	e.outageSeconds.Reset()
	for cause, seconds := range totals {
		e.outageSeconds.WithLabelValues(cause).Set(seconds)
	}
	e.outageSeconds.Collect(ch)
}

func (e *Exporter) emitDropRatio(ch chan<- prometheus.Metric, status *dynamic.Message, history []float64) {
	if drop, ok := floatField(status, "pop_ping_drop_rate"); ok {
		emit(ch, dishPopPingDropRatio, drop)
		return
	}
	if drop, ok := last(history); ok {
		emit(ch, dishPopPingDropRatio, drop)
	}
}

func (e *Exporter) emitObstruction(ch chan<- prometheus.Metric, stats *dynamic.Message) {
	if stats == nil {
		return
	}
	if obstructed, ok := boolField(stats, "currently_obstructed"); ok {
		emit(ch, dishCurrentlyObstructed, gaugeBool(obstructed))
	}
	emitFloat(ch, dishFractionObstruction, stats, "fraction_obstructed")
	emitFloat(ch, dishValidSeconds, stats, "valid_s")
	emitFloat(ch, dishObstructionTime, stats, "time_obstructed")
	emitFloat(ch, dishObstructionPatches, stats, "patches_valid")
	emitFloat(ch, dishProlongedDuration, stats, "avg_prolonged_obstruction_duration_s")
	emitFloat(ch, dishProlongedInterval, stats, "avg_prolonged_obstruction_interval_s")

	for name, series := range wedgeSeries(stats) {
		desc := dishWedgeFraction
		if containsAbs(name) {
			desc = dishWedgeAbsFraction
		}
		for i, v := range series {
			ch <- prometheus.MustNewConstMetric(
				desc, prometheus.GaugeValue, v,
				strconv.Itoa(i),
				fmt.Sprintf("%d_to_%d", i*30, (i+1)*30),
			)
		}
	}
}

func (e *Exporter) emitAlerts(ch chan<- prometheus.Metric, alerts *dynamic.Message) {
	flags := boolFlags(alerts)
	e.alerts.Reset()
	for name, val := range flags {
		e.alerts.WithLabelValues(name).Set(val)
		if desc := legacyAlerts[name]; desc != nil {
			emit(ch, desc, val)
		}
	}
	e.alerts.Collect(ch)
}

func (e *Exporter) emitReady(ch chan<- prometheus.Metric, ready *dynamic.Message) {
	flags := boolFlags(ready)
	e.ready.Reset()
	for name, val := range flags {
		e.ready.WithLabelValues(name).Set(val)
	}
	e.ready.Collect(ch)
}

func emitActuators(ch chan<- prometheus.Metric, status *dynamic.Message) {
	switch enumName(status, "has_actuators") {
	case "HAS_ACTUATORS_YES":
		emit(ch, dishHasActuators, 1)
	case "HAS_ACTUATORS_NO":
		emit(ch, dishHasActuators, 0)
	}
}

func emit(ch chan<- prometheus.Metric, desc *prometheus.Desc, value float64) {
	ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, value)
}

func emitFloat(ch chan<- prometheus.Metric, desc *prometheus.Desc, m *dynamic.Message, name string) {
	if v, ok := floatField(m, name); ok {
		emit(ch, desc, v)
	}
}

func emitBool(ch chan<- prometheus.Metric, desc *prometheus.Desc, m *dynamic.Message, name string) {
	if v, ok := boolField(m, name); ok {
		emit(ch, desc, gaugeBool(v))
	}
}

func containsAbs(name string) bool {
	return len(name) >= 3 && (name[:3] == "abs" || containsWord(name, "abs"))
}

func containsWord(name, word string) bool {
	for i := 0; i+len(word) <= len(name); i++ {
		if name[i:i+len(word)] == word {
			return true
		}
	}
	return false
}
