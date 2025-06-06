package cmd

import "fmt"

func Status() {
	//this needs to be looking at the sqllite
	//where the status for the router engine status needs to be updated
	statutsFromTheState := "Schema Validation issue with some sub schema"
	var status = "Working"
	if statutsFromTheState != "Working" {
		status = statutsFromTheState
	}
	fmt.Printf("Current KastQl status: %s \n", status)
}
