package main

import (
	"fmt"

	"github.com/Fl0k3n/k8s-inc/libs/p4-connector/connector"
)

func main() {
	con := connector.NewP4RuntimeConnector("0.0.0.0:9559", 0)
	fmt.Println(con)
}
