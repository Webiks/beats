package hardware

import (
	"log"
	"strconv"

	"github.com/StackExchange/wmi"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/system/hardware/util"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("system", "hardware", New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	hardwareQuery        []queryKey
	formatQuery          util.ConfigYaml
	hardware             common.MapStr
	hardwareMonitorQuery []queryKey
	config               util.ConfigYaml
}

type queryKey struct {
	Type              string
	Name              string
	DeviceID          string
	Description       string
	Manufacturer      string
	UserFriendlyName  []int8
	YearOfManufacture int
	Output            util.InnerConfigFormat
	Index             int
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The system hardware metricset by Webiks is beta v0.0.8 - 2020-05-24")
	config := struct{}{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}
	var newQuery = []queryKey{}
	var monitorQuery = []queryKey{}
	var cfg util.ConfigYaml
	util.ReadFile(&cfg)

	return &MetricSet{
		BaseMetricSet:        base,
		hardwareQuery:        newQuery,
		hardwareMonitorQuery: monitorQuery,
		hardware:             common.MapStr{},
		config:               cfg,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {

	hardwareQuery, hardwareMonitorQuery := getHardwareQueries(m.config)
	metricSetFields := common.MapStr{}
	buildAndSendHardwareQueryEvent(hardwareQuery, report, metricSetFields, false)
	buildAndSendHardwareQueryEvent(hardwareMonitorQuery, report, metricSetFields, true)

	if len(metricSetFields) > 0 {
		var event mb.Event
		event.MetricSetFields = metricSetFields
		report.Event(event)
	}

	return nil
}

func sendEventHardware(hard queryKey, hardware []queryKey, rootFields common.MapStr, metricSetFields common.MapStr, report mb.ReporterV2) {
	if hard.Output.UseConst == true {
		report.Event(mb.Event{
			MetricSetFields: common.MapStr{
				"data": rootFields,
			},
		})
	}
	if hard.Output.UseType == true {
		if len(hardware) == 1 {
			metricSetFields[hard.Type] = rootFields
		} else if hard.Index == 1 {
			metricSetFields[hard.Type] = common.MapStr{
				strconv.Itoa(hard.Index): rootFields,
			}
		} else {
			newMap := metricSetFields[hard.Type].(common.MapStr)
			newMap[strconv.Itoa(hard.Index)] = rootFields
		}
	}
}

func getHardwareQueries(cfg util.ConfigYaml) ([]queryKey, []queryKey) {
	var hardwareQuery = []queryKey{}
	var hardwareMonitorQuery = []queryKey{}

	for _, value := range cfg.Query {
		if value.TypeOf != "WmiMonitorID" {
			var dst []queryKey
			wmi.Query("Select * from "+value.TypeOf, &dst)
			for i, v := range dst {
				hardwareQuery = append(hardwareQuery, queryKey{Name: v.Name, Description: v.Description, DeviceID: v.DeviceID, Manufacturer: v.Manufacturer, Type: value.Name, Output: cfg.Format, Index: i + 1})
			}
		} else {
			// Special ability to handle WmiMonitorID
			var dst []queryKey
			err := wmi.QueryNamespace("select * from "+value.TypeOf, &dst, "root\\WMI")
			if err != nil {
				log.Println(err)
			}
			for i, v := range dst {
				hardwareMonitorQuery = append(hardwareMonitorQuery, queryKey{UserFriendlyName: v.UserFriendlyName, YearOfManufacture: v.YearOfManufacture, Type: value.Name, Output: cfg.Format, Index: i + 1})
			}
		}
	}

	return hardwareQuery, hardwareMonitorQuery
}

func buildAndSendHardwareQueryEvent(Query []queryKey, report mb.ReporterV2, metricSetFields common.MapStr, isMonitor bool) {
	if isMonitor {
		for _, hard := range Query {
			rootFields := common.MapStr{
				"type":             hard.Type,
				"name":             util.B2s(hard.UserFriendlyName),
				"manufacturerYear": hard.YearOfManufacture,
				"index":            hard.Index,
			}
			sendEventHardware(hard, Query, rootFields, metricSetFields, report)
		}
	} else {
		for _, hard := range Query {
			rootFields := common.MapStr{
				"type":         hard.Type,
				"name":         hard.Name,
				"description":  hard.Description,
				"manufacturer": hard.Manufacturer,
				"deviceID":     hard.DeviceID,
				"index":        hard.Index,
			}
			sendEventHardware(hard, Query, rootFields, metricSetFields, report)
		}
	}
}
