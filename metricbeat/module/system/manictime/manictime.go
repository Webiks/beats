package manictime

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/StackExchange/wmi"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/yaml.v2"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("system", "manictime", New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	database *sql.DB
	userName string
}

type ManicTimeConfig struct {
	Path string `yaml:"path"`
	User string `yaml:"user"`
}

type Config struct {
	Settings ManicTimeConfig `yaml:"manictime_metricset_settings"`
}

type Activity struct {
	title     string
	startTime string
	endTime   string
	url       sql.NullString
	appKey    sql.NullString
	appName   sql.NullString
	siteKey   sql.NullString
	siteName  sql.NullString
}

type ValidActivity struct {
	title           string
	startTime       string
	endTime         string
	url             string
	appKey          string
	appName         string
	siteKey         string
	siteName        string
	id              string
	durationMin     float64
	durationSec     float64
	applicationName string
}

type Win32_ComputerSystem struct {
	UserName string
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The system manictime metricset by Webiks is a beta v0.0.9.1 - 2020-08-03")
	config := struct{}{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	// read config
	var cfg Config
	readFile(&cfg)

	// connect to db
	database, err := sql.Open("sqlite3", cfg.Settings.Path)
	if err != nil {
		fmt.Println("could not open db file of manicTime")
	}
	// get current username
	userName := getUserName(cfg)
	return &MetricSet{
		BaseMetricSet: base,
		database:      database,
		userName:      userName,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {

	lastSync := getLastSyncTime(m.database)
	// parse string(date) to time
	parsedLastSync, _ := time.Parse(time.RFC3339, lastSync)

	// get all data
	newData := getManicTimeNewData(m.database, parsedLastSync)

	for _, activity := range newData {
		rootFields := common.MapStr{
			"title":           activity.title,
			"startTime":       activity.startTime,
			"endTime":         activity.endTime,
			"url":             activity.url,
			"appKey":          activity.appKey,
			"appName":         activity.appName,
			"siteKey":         activity.siteKey,
			"siteName":        activity.siteName,
			"durationMin":     activity.durationMin,
			"durationSec":     activity.durationSec,
			"applicationName": activity.applicationName,
			"id":              activity.id,
		}
		report.Event(mb.Event{
			MetricSetFields: common.MapStr{
				"data": rootFields,
			},
			RootFields: common.MapStr{
				"user": common.MapStr{
					"name": m.userName,
				},
			},
		})
	}

	//     update lastsync (memory + table)
	updateLastSync(m.database)

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

func getLastSyncTime(database *sql.DB) string {
	_, tableErr := database.Query(`SELECT lastSync FROM Sync WHERE id=1`)

	if tableErr != nil {
		fmt.Println("no date table or date table error", tableErr)
		statement, _ := database.Prepare("CREATE TABLE Sync(id INT PRIMARY KEY,lastSync datetime)")
		statement.Exec()
		loc, _ := time.LoadLocation("UTC")
		now := time.Now().In(loc)
		insertTimeStatement, _ := database.Prepare("INSERT INTO Sync (id, lastSync) VALUES (?,?)")
		fmt.Println(now, "date and time now")
		insertTimeStatement.Exec(1, now)
	}

	var lastSync string
	getLasySync, _ := database.Query("SELECT lastSync FROM Sync WHERE id=1")

	for getLasySync.Next() {
		getLastSyncErr := getLasySync.Scan(&lastSync)
		if getLastSyncErr != nil {
			fmt.Println(getLastSyncErr)
		}
	}

	return lastSync
}

func getManicTimeNewData(database *sql.DB, lastTimeSync time.Time) []ValidActivity {

	var newData []ValidActivity

	rows, _ := database.Query(`SELECT a.Name as title, a.StartUtcTime as startTime, a.EndUtcTime as endTime, b.Name as url, c.Key as appKey, c.Name as appName, d.Key as siteKey, d.Name as siteName
	FROM "Ar_Activity" as a
	LEFT JOIN "Ar_Activity" as b
	ON a.ActivityId = b.RelatedActivityId
	LEFT JOIN "Ar_CommonGroup" as c
	ON a.CommonGroupId = c.CommonId
	LEFT JOIN "Ar_CommonGroup" as d
	ON b.CommonGroupId = d.CommonId
	WHERE a.RelatedActivityId is NULL`)

	for rows.Next() {
		p := Activity{}
		err := rows.Scan(&p.title, &p.startTime, &p.endTime, &p.url, &p.appKey, &p.appName, &p.siteKey, &p.siteName)
		if err != nil {
			fmt.Println("error scanning rows in manictime metricset", err)
		}

		if p.title == "Active" {
			continue
		}
		parsedEndTime, _ := time.Parse(time.RFC3339, p.endTime)
		if parsedEndTime.After(lastTimeSync) {
			newActivity := ValidActivity{}
			newActivity.title = p.title
			newActivity.startTime = getMaxStartTime(p.startTime, lastTimeSync)
			newActivity.endTime = p.endTime
			if p.url.Valid {
				newActivity.url = p.url.String
			} else {
				newActivity.url = ""
			}
			if p.appKey.Valid {
				newActivity.appKey = p.appKey.String
			} else {
				newActivity.appKey = ""
			}
			if p.appName.Valid {
				newActivity.appName = p.appName.String
			} else {
				newActivity.appName = ""
			}
			if p.siteKey.Valid {
				newActivity.siteKey = p.siteKey.String
			} else {
				newActivity.siteKey = ""
			}
			if p.siteName.Valid {
				newActivity.siteName = p.siteName.String
				newActivity.applicationName = p.siteName.String
			} else {
				newActivity.siteName = ""
				newActivity.applicationName = newActivity.appName
			}
			startTime, _ := time.Parse(time.RFC3339, newActivity.startTime)
			endTime, _ := time.Parse(time.RFC3339, newActivity.endTime)
			newActivity.durationMin = toFixed(endTime.Sub(startTime).Minutes(), 3)
			newActivity.durationSec = endTime.Sub(startTime).Seconds()
			hostname, _ := os.Hostname()
			id := newActivity.appName + "_" + newActivity.startTime + "_" + hostname
			newActivity.id = strings.ReplaceAll(id, " ", "_")
			newData = append(newData, newActivity)
		}
	}
	return newData
}

func updateLastSync(database *sql.DB) {
	loc, _ := time.LoadLocation("UTC")
	now := time.Now().In(loc)
	database.Exec("UPDATE Sync SET lastSync = $1 WHERE id=1", now)
}

func getMaxStartTime(startTime string, lastTimeSync time.Time) string {
	parsedStartTime, _ := time.Parse(time.RFC3339, startTime)
	if lastTimeSync.After(parsedStartTime) {
		return lastTimeSync.Format(time.RFC3339)
	}
	return parsedStartTime.Format(time.RFC3339)
}

func getUserName(cfg Config) string {
	var username string
	var query []Win32_ComputerSystem

	q := wmi.CreateQuery(&query, "")

	wmi.Query(q, &query)

	if query[0].UserName != "" {
		username = query[0].UserName
	} else {
		username = cfg.Settings.User
	}

	return username
}

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}
