/*
* @Author: Jim Weber
* @Date:   2016-08-10 17:43:45
* @Last Modified by:   Jim Weber
* @Last Modified time: 2016-08-11 19:14:44
 */

package main

// UnitFile a struct to hold the different types of unit files
// for container / application deployments.
type UnitFile struct {
	Contents string
	Type     string
	Global   bool
}
