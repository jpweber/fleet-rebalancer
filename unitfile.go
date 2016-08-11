/*
* @Author: Jim Weber
* @Date:   2016-08-10 17:43:45
* @Last Modified by:   Jim Weber
* @Last Modified time: 2016-08-11 19:54:32
 */

package main

// UnitFile a struct to hold the different types of unit files
// for container / application deployments.
type UnitFiles struct {
	File []File
}

type File struct {
	Name     string
	Contents string
	NewName  string
}
