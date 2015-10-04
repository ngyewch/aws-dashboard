package main

import ()

func main() {
	config, err := readConfig("aws-dashboard.yaml")
	if err != nil {
		panic(err)
	}

	processOverview(config)
	processBilling(config)
}
