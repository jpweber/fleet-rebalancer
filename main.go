/*
* @Author: Jim Weber
* @Date:   2016-08-10 17:43:45
* @Last Modified by:   Jim Weber
* @Last Modified time: 2016-08-10 20:13:22
 */

package main

import (
	"flag"
	"log"
)

func main() {

	fleetHost := flag.String("f", "", "Fleet host to send commands to <hostname>:<port>")
	machineID := flag.String("m", "", "Machine ID to reschedule away from")
	debug := flag.Bool("v", false, "verbose output")
	flag.Parse()

	// fleetHost := "coreos.dev.crosschx.com:49153"    // TODO: will come from cli args
	// machineID := "2d69b20e090a4859b2c9ec7d48b0188c" // TODO: will come from cli args

	log.Println("Starting Fleet Rescheduling")

	unitStates := instanceStates(*fleetHost)
	unitCount := len(unitStates.States)
	machines := machineCount(unitStates)
	countOnMachine := containerCount(unitStates, *machineID)
	reschedule := countToReschedule(unitCount, machines, countOnMachine)
	movingUnits := unitsToReschule(reschedule, unitStates)
	if *debug == true {
		log.Println(unitCount, "Containers")
		log.Println(machines, "Fleet Nodes")
		log.Println(countOnMachine, "Containers on node we want to cleanup")
		log.Println(reschedule, "Number of containers we are going to reschedule away from", *machineID)
		log.Println("Units that are going to be moved", movingUnits)
	}

}
