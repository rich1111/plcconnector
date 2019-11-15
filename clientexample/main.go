package main

import (
	"fmt"

	plc ".."
)

func main() {
	err := plc.Discover("192.168.1.255:44818")
	if err != nil {
		fmt.Println(err)
	}
}
