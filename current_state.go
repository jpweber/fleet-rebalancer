/*
* @Author: Jim Weber
* @Date:   2016-08-10 17:43:45
* @Last Modified by:   Jim Weber
* @Last Modified time: 2016-08-10 20:02:35
 */

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
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

func instanceStates(fleetHost string) FleetStates {
	url := "http://" + fleetHost + "/fleet/v1/state"
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
