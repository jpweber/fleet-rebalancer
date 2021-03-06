/*
* @Author: Jim Weber
* @Date:   2016-08-10 17:43:45
* @Last Modified by:   Jim Weber
* @Last Modified time: 2016-10-20 18:00:01
 */

// TODO: Automatically find the node with the most containers

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

// DeployInfo struct to hold the deployment information
type DeployInfo struct {
	Version  string
	AppName  string
	UnitFile string
}

// Hosts hold host info
type Hosts struct {
	fleet string
	etcd  string
}

func main() {

	fleetHost := flag.String("f", "", "Fleet Host")
	etcdHost := flag.String("e", "", "Etcd Host")
	// machineID := flag.String("m", "", "Machine ID to reschedule away from")
	debug := flag.Bool("v", false, "verbose output")
	dryRun := flag.Bool("d", false, "Dry Run. Don't make any changes just show what would happen")
	flag.Parse()

	// define our hosts in the host struct
	hosts := Hosts{
		fleet: *fleetHost,
		etcd:  *etcdHost,
	}

	// if we are in try dry run mode automatically enable debug mode
	if *dryRun == true {
		*debug = true
	}

	log.Println("Starting Fleet Rescheduling")

	unitStates := instanceStates(hosts, nil)
	unitCount := len(unitStates.States)
	machines := machineCount(unitStates)
	highCountHost := mostContainers(unitStates)
	countOnMachine := containerCount(unitStates, highCountHost)
	reschedule := countToReschedule(unitCount, machines, countOnMachine)
	movingUnits := unitsToReschule(reschedule, unitStates, highCountHost)
	destroyingUnits := unitsToDestroy(reschedule, unitStates, highCountHost)

	if *debug == true {
		log.Println(highCountHost, "Host with highest count of containers")
		log.Println(unitCount, "Containers")
		log.Println(machines, "Fleet Nodes")
		log.Println(countOnMachine, "Containers on node we want to cleanup")
		log.Println(reschedule, "Number of containers we are going to reschedule away from", highCountHost)
		log.Println("Units that are going to be moved", movingUnits)
		log.Println("Units that are going to be destroyed", destroyingUnits)
	}
	// early exit conditionals
	if *dryRun == true || reschedule == 0 {
		os.Exit(0)
	}

	unitFiles := UnitFiles{}
	for _, unitName := range destroyingUnits {
		unitFile := getUnitfile(unitName, hosts)
		file := File{
			Name:     unitName,
			Contents: unitFile,
		}
		unitFiles.File = append(unitFiles.File, file)
	}

	appDeployData := []DeployInfo{}

	for _, unit := range movingUnits {
		deployInfo := DeployInfo{}
		var unitFile string
		appName := nameForNextInstance(unit)
		versioNumber := getAppVersionNumber(unit)

		// find the right unit file not a great way but it works.
		for _, file := range unitFiles.File {
			if strings.Contains(file.Name, unit) {
				unitFile = file.Contents
			}
		}
		deployInfo.AppName = appName
		deployInfo.Version = versioNumber
		deployInfo.UnitFile = unitFile

		appDeployData = append(appDeployData, deployInfo)
	}

	for _, deployData := range appDeployData {
		deployResults := deployUnits(hosts, deployData.AppName, deployData.Version, deployData.UnitFile)
		if deployResults == true {

			// if the container succesfully deployed destroy
			// all old instances of this container

			for _, killUnit := range destroyingUnits {
				if strings.Contains(killUnit, deployData.AppName) {
					destroyInstance(killUnit, hosts)
				}
			}
		} else {
			fmt.Println("Failed rescheduling container. Going on to next one")
		}
	}

}
