/*
* @Author: Jim Weber
* @Date:   2016-05-18 22:10:02
* @Last Modified by:   Jim Weber
* @Last Modified time: 2016-08-10 22:29:34
 */

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/etcd/client"
	"github.com/fatih/color"
	"golang.org/x/net/context"
)

func nameForNextInstance(unit string) string {
	rx := regexp.MustCompile("[0-9]+")

	nameParts := strings.Split(unit, "-")
	nameLimit := 0
	for idx, part := range nameParts {
		if rx.MatchString(part) == true {
			nameLimit = idx - 1
			break
		}
	}

	appName := strings.Join(nameParts[0:3], "-")
	return appName
}

func getNextInstance(host string, appName string) int64 {
	// TODO: update this to use the etcd library
	url = "http://" + host + "/v2/keys/nextinstance/" + appName
	var curInstance int64
	var nextInstanceNum int64
	response, err := http.Get(url)
	if err != nil {
		fmt.Printf("%s", err)
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Printf("%s", err)
		}

		var etcdResp map[string]interface{}
		if err := json.Unmarshal(contents, &etcdResp); err != nil {
			panic(err)
		}

		if etcdResp["errorCode"] != nil {
			fmt.Println("Instance number does not exist. Initializing new key")
			// initialize instance key
			setInstanceNumber(deployInfo, 10, 0)
			nextInstanceNum = 10
		} else {
			nodeResp := etcdResp["node"].(map[string]interface{})
			instanceValue := nodeResp["value"].(string)
			curInstance, _ = strconv.ParseInt(instanceValue, 0, 0)
			nextInstanceNum = curInstance + 1
			if nextInstanceNum >= 99 {
				// we are purposely not going over 99
				// always reset to 10 if we are at 99 or greater because of a bug
				nextInstanceNum = 10
			}
			setInstanceNumber(host, nextInstanceNum, curInstance)
		}

	}

	return nextInstanceNum
}

func setInstanceNumber(host string, appName string, instanceNumber int64, prevValue int64) {
	url := "http://" + host
	cfg := client.Config{
		Endpoints: []string{fleetURL + ":4001"},
		Transport: client.DefaultTransport,
		// set timeout per request to fail fast when the target endpoint is unavailable
		HeaderTimeoutPerRequest: time.Second,
	}
	etcdClient, err := client.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	kapi := client.NewKeysAPI(etcdClient)
	setOptions := client.SetOptions{}
	if prevValue != 0 {
		setOptions.PrevValue = fmt.Sprintf("%d", prevValue)
	}
	_, err = kapi.Set(context.Background(), "/nextinstance/"+appName, fmt.Sprintf("%d", instanceNumber), &setOptions)
	if err != nil {
		log.Fatal(err)
	}
}

func handleInstanceTimeout(deployInfo DeployInfo, instanceNumber string) {
	// get instance state info from fleet before
	// printing error and moving on
	color.Red("Timeout waiting for container to be up")
	fleetParams := map[string]string{"unitName": deployInfo.AppName + "-" + deployInfo.Version + "@" + instanceNumber}
	fleetResp := getInstanceStates(deployInfo, fleetParams)
	if len(fleetResp.States) > 0 {
		fmt.Println("systemdActiveState:", fleetResp.States[0].SystemdActiveState)
		fmt.Println("systemdLoadState:", fleetResp.States[0].SystemdLoadState)
		fmt.Println("systemdSubState:", fleetResp.States[0].SystemdSubState)
	} else {
		instanceName := deployInfo.AppName + "-" + deployInfo.Version + "@" + instanceNumber
		fmt.Println(instanceName, "Not in list of fleet units.")
		resp := getInstanceStates(deployInfo, nil)
		deployedUnits := filterInstances(resp, deployInfo)
		numNodes := getNumberOfNodes(deployInfo)

		if numNodes == len(deployedUnits) {
			fmt.Println(numNodes, "Nodes in Cluster", len(deployedUnits), "Units deployed")
			fmt.Println("at least one unit must be destroy first before we can deploy a new one")
		}

	}
}

func instanceUp(deployInfo DeployInfo, instanceNumber string, waitSecs int) bool {
	var up bool
	etcdURL := "http://coreos." + deployInfo.Environ + ".crosschx.com"
	cfg := client.Config{
		Endpoints: []string{etcdURL + ":4001"},
		Transport: client.DefaultTransport,
		// set timeout per request to fail fast when the target endpoint is unavailable
		HeaderTimeoutPerRequest: time.Second,
	}
	etcdClient, err := client.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	kapi := client.NewKeysAPI(etcdClient)
	watchOptions := client.WatcherOptions{0, false}
	watcher := kapi.Watcher("/services/instances/"+deployInfo.AppName+"/"+deployInfo.Version+"@"+instanceNumber, &watchOptions)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(waitSecs)*time.Second)
	defer cancel()
	fmt.Println("Waiting", waitSecs, "seconds for container to be up")

	// start watching the fleet state of our unit / instance
	stateChan := make(chan string)
	quit := make(chan bool)
	go watchFleetState(deployInfo, instanceNumber, stateChan, quit)
	go func() {
		for {
			select {
			case state := <-stateChan:
				if state == "failed" {
					log.Println("Unit State", state)
					fmt.Println("Fleet unit entered failed state")
					quit <- true
					cancel()
					return
				} else {
					log.Println("Unit State", state)
				}
			}

		}
	}()

	_, err = watcher.Next(ctx)
	if err != nil {
		if err == context.Canceled {
			// ctx is canceled by another routine
		} else if err == context.DeadlineExceeded {
			handleInstanceTimeout(deployInfo, instanceNumber)
			up = false
		} else {
			// handle error
			up = false
		}
	} else {
		up = true
		quit <- true
	}

	return up
}
