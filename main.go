/*
* @Author: Jim Weber
* @Date:   2016-08-10 17:43:45
* @Last Modified by:   Jim Weber
* @Last Modified time: 2016-08-11 19:20:20
 */

package main

import (
	"flag"
	"fmt"
	"log"
)

// DeployInfo struct to hold the deployment information
type DeployInfo struct {
	Version  string
	AppName  string
	UnitFile string
}

func main() {

	fleetHost := flag.String("f", "", "Fleet host to send commands to <hostname>:<port>")
	machineID := flag.String("m", "", "Machine ID to reschedule away from")
	debug := flag.Bool("v", false, "verbose output")
	flag.Parse()

	log.Println("Starting Fleet Rescheduling")

	unitStates := instanceStates(*fleetHost, nil)
	unitCount := len(unitStates.States)
	machines := machineCount(unitStates)
	countOnMachine := containerCount(unitStates, *machineID)
	reschedule := countToReschedule(unitCount, machines, countOnMachine)
	movingUnits := unitsToReschule(reschedule, unitStates)
	destroyingUnits := unitsToDestroy(reschedule, unitStates)
	if *debug == true {
		log.Println(unitCount, "Containers")
		log.Println(machines, "Fleet Nodes")
		log.Println(countOnMachine, "Containers on node we want to cleanup")
		log.Println(reschedule, "Number of containers we are going to reschedule away from", *machineID)
		log.Println("Units that are going to be moved", movingUnits)
		log.Println("Units that are going to be destroyed", destroyingUnits)
	}

	// TODO: deploy unit
	deployResults := deployUnits(deployInfo, unitFiles)
	if deployResults == true {
		// decrement total number of expected deployed containers
		// *numContainers--

		// if the container succesfully deployed destroy
		// all old instances of this container
		// loop over the oldInstances and send a destroy command
		// for each one, run this as goroutines so they operate conncurrently
		// for oldInstanceCount > 0 {
		//  fmt.Println(oldInstanceCount)
		//  go destroyInstance(oldInstances[oldInstanceCount-1], deployInfo)
		//  oldInstanceCount--
		// }
	} else {
		fmt.Println("Failed rescheduling container. Going on to next one")
	}

}
