package chrome_extension

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("system", "chrome_extension", New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.

type ChromeExtensionConfig struct {
	Port      string `yaml:"port"`
	ShowInMin bool   `yaml:"showInMin"`
}

type Config struct {
	Settings ChromeExtensionConfig `yaml:"chrome_extension_metricset_settings"`
}

type URLData struct {
	Domain string  `json:"domain"`
	Time   float32 `json:"time"`
}

type MetricSet struct {
	mb.BaseMetricSet
	data   common.MapStr
	config Config
}

var allData = make([]URLData, 0)

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The system chrome_extension metricset by Webiks is a beta v0.0.9 - 2020-07-22")

	config := struct{}{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	// read config
	var cfg Config
	readFile(&cfg)

	// init server
	r := mux.NewRouter()
	// route handlers/endpoints
	r.HandleFunc("/ok", testIsServerOk).Methods("GET")

	r.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		sendURLData(w, r, &allData)
	}).Methods("POST")

	r.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		getURLData(w, r, &allData)
	}).Methods("GET")

	// running the server
	srv := &http.Server{
		Addr: "0.0.0.0:" + cfg.Settings.Port,
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r, // Pass our instance of gorilla/mux in.
	}
	// Run our server in a goroutine so that it doesn't block.
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	return &MetricSet{
		BaseMetricSet: base,
		data:          common.MapStr{},
		config:        cfg,
	}, nil

}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	for _, domain := range allData {
		rootFields := common.MapStr{
			"domain":    domain.Domain,
			"timeInSec": domain.Time,
		}
		if m.config.Settings.ShowInMin {
			rootFields["timeInMin"] = domain.Time / 60
		}
		report.Event(mb.Event{
			MetricSetFields: common.MapStr{
				"data": rootFields,
			},
		})
	}
	resetAllData(&allData)
	return nil
}

// read config file
func readFile(cfg *Config) {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	filename, _ := filepath.Abs(exPath + `\metricbeat.yml`)
	yamlFile, err := ioutil.ReadFile(filename)

	if err != nil {
		fmt.Println(err)
	}

	err = yaml.Unmarshal(yamlFile, &cfg)
	if err != nil {
		fmt.Println(err)
	}

}

// Test is server ok Route
func testIsServerOk(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode("server is ok")
}

// Post UrlData Route
func sendURLData(w http.ResponseWriter, r *http.Request, allData *[]URLData) {
	w.Header().Set("Content-Type", "application/json")
	var data []URLData
	_ = json.NewDecoder(r.Body).Decode(&data)

	for _, item := range data {
		*allData = append(*allData, item)
	}

	json.NewEncoder(w).Encode(data)
}

// Get UrlData Route
func getURLData(w http.ResponseWriter, r *http.Request, allData *[]URLData) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(*allData)

}

func resetAllData(allData *[]URLData) {
	*allData = make([]URLData, 0)
}
