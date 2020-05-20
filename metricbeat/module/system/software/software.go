package software

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"

	"golang.org/x/sys/windows/registry"
	"gopkg.in/yaml.v2"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("system", "software", New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
// add struct for  get data - webiks added

type Config struct {
	Software []string `yaml:"software"`
}

type Software struct {
	Name         string
	Version      string
	MajorVersion uint64
	MinorVersion uint64
	Date         string
}

type MetricSet struct {
	mb.BaseMetricSet
	softwares []Software
	software  common.MapStr
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The system software metricset by webiks is beta, version 0.0.5")

	config := struct{}{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		software:      common.MapStr{},
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the  right
// format. It publishes the event which is then forwarded to the output. I n case
// of an error set the Error field of mb.Event or simply call report.Error ().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {

	log.Println("---software log---, before reading config file")
	// read config
	var cfg Config
	// get current directory

	readFile(&cfg)

	log.Println("---software log---, before reading registery")
	// get info from registery
	var data = readAllSoftwareRegistery()

	log.Println("---software log---, before filtering softwares")
	// filter data by query
	var filterdData = filterByYmlField(data, cfg.Software)

	for _, soft := range filterdData {
		rootFields := common.MapStr{
			"name":         soft.Name,
			"version":      soft.Version,
			"majorVersion": soft.MajorVersion,
			"minorVersion": soft.MinorVersion,
			"InstallDate":  soft.Date,
		}
		log.Println("---software log---, before sending event")
		report.Event(mb.Event{
			MetricSetFields: common.MapStr{

				"data": rootFields,
			},
		})
	}
	return nil
}

// registery software functions

func readFile(cfg *Config) {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	fmt.Println(exPath, "exec path")

	filename, _ := filepath.Abs(exPath + `\software.yml`)
	yamlFile, err := ioutil.ReadFile(filename)

	if err != nil {
		fmt.Println(err)
	}

	err = yaml.Unmarshal(yamlFile, &cfg)
	if err != nil {
		fmt.Println(err)
	}

}

func readRegistry(path string) []Software {
	log.Println("---software log---inside readRegisry, open key to unistall reg")
	// get key from registery
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.READ)
	if err != nil {
		log.Fatal(err)
		log.Println("---software log---inside readRegisry, error open key")
	}
	defer k.Close()

	log.Println("---software log---inside readRegisry, read subkeys")
	// read all subskeys of uninstall
	s, err := k.ReadSubKeyNames(0)
	if err != nil {
		log.Fatal(err)
		log.Println("---software log---inside readRegisry, error read sub key")
	}

	var value string
	var data []Software
	for _, value = range s {
		log.Println("---software log---inside readRegisry,open key to each subkey")
		// open key for each
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, path+`\`+value, registry.READ)
		if err != nil {
			log.Fatal(err)
			log.Println("---software log---inside readRegisry, error open key to sub key")
		}
		defer k.Close()
		// from each key get values of display name and display version
		log.Println("---software log---inside readRegisry,getting value of display name,version,install date ")
		displayName, _, err := k.GetStringValue("DisplayName")
		displayVersion, _, err := k.GetStringValue("DisplayVersion")
		installDate, _, err := k.GetStringValue("InstallDate")

		// get install year,month,day
		dateString := ""

		if len(installDate) == 8 {
			fulldate := []string{installDate[:4], installDate[4:6], installDate[6:8]}
			dateString = strings.Join(fulldate, "/")
		} else {
			dateString = "Unknown"
		}

		// split display version into array by dot
		versionSplitted := strings.Split(displayVersion, ".")

		if len(versionSplitted) > 1 {
			majorVersion := versionSplitted[0]
			minorVersion := versionSplitted[1]
			// convert the mintor and major strings to int
			majorVersionInt, _ := strconv.ParseUint(majorVersion, 10, 64)
			minorVersionInt, _ := strconv.ParseUint(minorVersion, 10, 64)
			log.Println("---software log---inside readRegisry,creating instanch of software with all details ")
			// creating the instance of software object
			newData := Software{Name: displayName, Version: displayVersion, MajorVersion: majorVersionInt, MinorVersion: minorVersionInt, Date: dateString}
			if displayName != "" {
				if displayVersion != "" {
					data = append(data, newData)
				}
			}
		}
	}
	return data
}

func readAllSoftwareRegistery() []Software {
	var combinedArray []Software
	var win32reg = readRegistry(`Software\Microsoft\Windows\CurrentVersion\Uninstall`)
	var win64reg = readRegistry(`SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall`)
	combinedArray = append(combinedArray, win32reg...)
	combinedArray = append(combinedArray, win64reg...)
	return combinedArray
}

func filterByYmlField(data []Software, query []string) []Software {
	var filterdArray []Software

	for _, value := range query {
		for _, software := range data {
			value = string(strings.ToUpper(value))
			software.Name = string(strings.ToUpper(software.Name))
			// fmt.Println(software.Name, value
			if strings.HasPrefix(software.Name, value) {
				// check if the item allready been insert before
				counter := 0
				for i := range filterdArray {
					if filterdArray[i].Name == software.Name {
						counter++
						break
					}
				}
				// if its not included append it to filterdArray.
				if counter == 0 {
					filterdArray = append(filterdArray, software)
				}
			}
		}
	}
	return filterdArray
}
