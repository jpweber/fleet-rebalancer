/*
* @Author: Jim Weber
* @Date:   2016-08-10 17:43:45
* @Last Modified by:   Jim Weber
* @Last Modified time: 2016-09-30 00:47:53
 */

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"strings"
)

type FleetStates struct {
	States []struct {
		SystemdActiveState string `json:"systemdActiveState"`
		MachineID          string `json:"machineID"`
		Hash               string `json:"hash"`
		SystemdSubState    string `json:"systemdSubState"`
		Name               string `json:"name"`
		SystemdLoadState   string `json:"systemdLoadState"`
	}

	MachineCount    int
	CountainerCount int
}

func instanceStates(hosts Hosts, params map[string]string) FleetStates {
	url := "http://" + hosts.fleet + "/fleet/v1/state"
	// loop through params to append to the url if they exist
	if len(params) > 0 {
		url = url + "?"
		for key, value := range params {
			// as of now we are only ever expecting a single k,v pair
			// for parameters
			url = url + key + "=" + value + ".service"
		}
	}
	response, err := http.Get(url)
	fleetStates := FleetStates{}

	if err != nil {
		fmt.Printf("%s", err)
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Printf("%s", err)
			os.Exit(1)
		}

		if err := json.Unmarshal(contents, &fleetStates); err != nil {
			panic(err)
		}

	}

	return fleetStates

}

func containerCount(fleetUnits FleetStates, machineID string) int {
	containerCount := 0
	for _, fleetUnit := range fleetUnits.States {
		if fleetUnit.MachineID == machineID {
			containerCount++
		}
	}

	return containerCount
}

func machineCount(fleetUnits FleetStates) int {
	var machines = make(map[string]bool)
	for _, fleetUnit := range fleetUnits.States {
		machines[fleetUnit.MachineID] = true
	}

	return len(machines)
}

func countToReschedule(containers, machines, countOnMachine int) int {
	reschedule := float64(countOnMachine) - math.Ceil(float64(containers)/float64(machines))
	return int(reschedule)
}

func unitsToReschule(rescheduleCount int, fleetUnits FleetStates, machineID string) []string {
	var units []string
	if rescheduleCount == 0 {
		return units
	}
	idx := 0
	for _, fleetUnit := range fleetUnits.States {
		if fleetUnit.MachineID == machineID {
			idx++
			nameParts := strings.Split(fleetUnit.Name, "@")
			units = append(units, nameParts[0])
			if idx == rescheduleCount {
				break
			}
		}
	}

	return units
}

func unitsToDestroy(rescheduleCount int, fleetUnits FleetStates, machineID string) []string {
	var units []string
	if rescheduleCount == 0 {
		return units
	}
	idx := 0
	for _, fleetUnit := range fleetUnits.States {
		if fleetUnit.MachineID == machineID {
			idx++
			units = append(units, fleetUnit.Name)
			if idx == rescheduleCount {
				break
			}
		}
	}
	// for i := 0; i < rescheduleCount; i++ {
	// 	units = append(units, fleetUnits.States[i].Name)
	// }
	return units
}
