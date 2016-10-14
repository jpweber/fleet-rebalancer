/*
* @Author: Jim Weber
* @Date:   2016-05-18 22:07:31
* @Last Modified by:   Jim Weber
* @Last Modified time: 2016-09-27 23:12:46
 */

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/fatih/color"
)

func sendUnitFile(host, appName, instanceNumber, unitFile string) *http.Response {

	// using bytes buffer to concat string quickly
	var url bytes.Buffer
	url.WriteString("http://")
	url.WriteString(host)
	url.WriteString("/fleet/v1/units/")
	url.WriteString(appName)
	url.WriteString("@")
	url.WriteString(instanceNumber)
	url.WriteString(".service")

	// TODO: @debug
	fmt.Println(url.String())
	req, err := http.NewRequest("PUT", url.String(), bytes.NewBuffer([]byte(unitFile)))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 201 {
		fmt.Println("Fleet accepted the unit file OK")
	} else {
		// if fleet returns a response code other than ok show the code and body
		fmt.Println("response Status:", resp.Status)
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println("response Body:", string(body))
	}

	return resp

}

// wrapper function for sending unit file and other housekeeping parts of that
func deployUnits(hosts Hosts, appName, appVersion, unitFile string) bool {
	// get the next instance number to use for deployment
	// this function will also handling initializing the instance number
	// if one does not exist

	fmt.Println("getting instance number for non global unit")
	nextInstNum := getNextInstance(hosts.etcd, appName)

	// init what we are going to return
	var status bool

	// deploy new unit.
	sendTries := 5
	for sendTries != 0 {
		sendUnitResponse := sendUnitFile(hosts.fleet, appName+"-"+appVersion, fmt.Sprintf("%d", nextInstNum), unitFile)
		if sendUnitResponse.StatusCode != 201 {
			// special catch for 204 errors.
			if sendUnitResponse.StatusCode == 204 {
				color.Red("Received 204 - Duplicate unit file submitted to fleet. This usually a sign multiple unit files for this version. Contact DevOPs")
			} else {
				color.Red("Error communicating with fleet trying again")
			}
			sendTries--
			if sendTries == 0 {
				color.Red("Deployment Failed")
			}
			time.Sleep(1 * time.Second)
		} else {
			// we succeeded now break out of this loop
			sendTries = 0
		}
	}

	// now wait for the container to be up
	// only for the main unit types. Not watching for presence yet
	success := instanceUp(hosts, appName, appVersion, fmt.Sprintf("%d", nextInstNum), 600)
	if success == true {
		status = true
		color.Green("Deployment Successful")
	} else {
		status = false
	}

	// default to false but we should never really hit this
	return status

}

func destroyInstance(oldInstance string, hosts Hosts) {
	// first we need to set it to inactive when we can destroy it
	// this is because of a bug in fleet with systemd not executing
	// execstoppost actions https://github.com/coreos/fleet/issues/1000
	// url := "http://coreos." + deployInfo.Environ + ".crosschx.com:49153/fleet/v1/units/" + oldInstance
	// temporary hard coded for now
	url := "http://" + hosts.fleet + "/fleet/v1/units/" + oldInstance
	// stop
	fmt.Println("Stopping", oldInstance)
	stopState := `{"desiredState": "inactive"}`
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer([]byte(stopState)))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	// fmt.Println("response Headers:", resp.Header)

	// destroy
	fmt.Println("Destroying", oldInstance)
	req, err = http.NewRequest("DELETE", url, nil)
	req.Header.Set("Content-Type", "application/json")

	client = &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		fmt.Println(oldInstance, "Destroyed")
	}
}

func watchFleetState(hosts Hosts, appName, appVersion, instanceNumber string, c chan string, q chan bool) {
	for {
		select {
		case <-q:
			return
		default:
			fleetStateParams := map[string]string{"unitName": appName + "-" + appVersion + "@" + instanceNumber}
			state := instanceStates(hosts, fleetStateParams)
			if len(state.States) > 0 {
				c <- state.States[0].SystemdSubState
			}

			// sleep for .25 seconds to not DoS our fleet api
			time.Sleep(250 * time.Millisecond)
		}

	}
}

func getUnitfile(unitName string, hosts Hosts) string {
	url := "http://" + hosts.fleet + "/fleet/v1/units/" + unitName

	response, err := http.Get(url)
	var contents []byte
	var dat map[string]interface{}

	if err != nil {
		fmt.Printf("%s", err)
	} else {
		defer response.Body.Close()
		contents, err = ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Printf("%s", err)
			os.Exit(1)
		}

		if err := json.Unmarshal(contents, &dat); err != nil {
			panic(err)
		}

		// cleanupt our returned values to be what we really want
		delete(dat, "currentState")
		delete(dat, "machineID")
		delete(dat, "name")

	}

	returnUnit, _ := json.Marshal(dat)
	return string(returnUnit)

}
