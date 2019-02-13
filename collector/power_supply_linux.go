// +build !nopower_supply

package collector

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	registerCollector("power_supply", defaultEnabled, NewPowerSupplyCollector)
}

type powerSupplyCollector struct{}

func NewPowerSupplyCollector() (Collector, error) {
	return &powerSupplyCollector{}, nil
}

type valueDesc struct {
	key  string
	desc *prometheus.Desc
}

func makeValueDesc(name, help string) valueDesc {
	return valueDesc{
		name,
		prometheus.NewDesc("node_power_supply_" + name, help, []string{"power_supply"}, nil),
	}
}

var (
	infoKeys = []string{
		"power_supply",
		"status", // This probably shouldn't be here.
		"technology", "capacity_level",
		"model_name", "manufacturer", "serial_number",
	}
	infoDesc = prometheus.NewDesc("node_power_supply_info", "XXX", infoKeys, nil)

	valueDescs []valueDesc = []valueDesc{
		makeValueDesc("present",            "XXX"),
		makeValueDesc("online",             "XXX"),
		makeValueDesc("cycle_count",        "XXX"),
		makeValueDesc("voltage_min_design", "XXX"),
		makeValueDesc("voltage_now",        "XXX"),
		makeValueDesc("current_now",        "XXX"),
		makeValueDesc("charge_full_design", "XXX"),
		makeValueDesc("charge_full",        "XXX"),
		makeValueDesc("charge_now",         "XXX"),
		makeValueDesc("capacity",           "XXX"),
	}
)

func (c *powerSupplyCollector) updatePowerSupplyDir(ch chan<- prometheus.Metric, dirPath string) error {
	ueventPath := path.Join(dirPath, "uevent")
	file, err := os.Open(ueventPath)
	if err != nil {
		return err
	}
	defer file.Close()

	data := map[string]string{}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		eqIndex := strings.IndexByte(line, '=')
		if eqIndex == -1 {
			continue
		}
		key := strings.ToLower(strings.TrimPrefix(line[:eqIndex], "POWER_SUPPLY_"))
		value := line[eqIndex+1:]
		data[key] = value
	}

	name, ok := data["name"]
	if !ok {
		return fmt.Errorf("couldn't find name in %s", ueventPath)
	}

	infoValues := make([]string, 0, len(infoKeys))
	for _, key := range infoKeys {
		value := data[key]
		if key == "power_supply" {
			value = name
		}
		infoValues = append(infoValues, value)
	}
	ch <- prometheus.MustNewConstMetric(infoDesc, prometheus.UntypedValue, 1, infoValues...)

	for _, vd := range valueDescs {
		strVal, ok := data[vd.key]
		if !ok {
			continue
		}
		floatVal, err := strconv.ParseFloat(strVal, 64)
		if err != nil {
			continue
		}
		ch <- prometheus.MustNewConstMetric(vd.desc, prometheus.GaugeValue, floatVal, name)
	}

	return nil
}

func (c *powerSupplyCollector) Update(ch chan<- prometheus.Metric) error {
	powerSupplyPathName := path.Join(sysFilePath("class"), "power_supply")

	powerSupplyFiles, err := ioutil.ReadDir(powerSupplyPathName)
	if err != nil {
		return err
	}

	for _, dir := range powerSupplyFiles {
		dirPath := path.Join(powerSupplyPathName, dir.Name())
		if lastErr := c.updatePowerSupplyDir(ch, dirPath); lastErr != nil {
			err = lastErr
		}
	}

	return err
}
