package main

import (
	"github.com/KyleBanks/depth"
	"github.com/Nerzal/gocense"
)

func main() {
	gocenseClient := gocense.New()

	deps := gocenseClient.Get("github.com/swaggo/swag")
	printDeps(deps)

	mappings, err := gocenseClient.GetAllLicenses(deps)
	if err != nil {
		println(err.Error())
	}
}

func printDeps(deps []depth.Pkg) {
	for i := range deps {
		println(deps[i].Name)
	}
}
