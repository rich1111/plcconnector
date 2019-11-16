package main

import (
	"fmt"

	plc ".."
)

func main() {
	ids, err := plc.Discover("192.168.1.255:44818")
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, id := range ids {
		fmt.Printf("%#v\n", id)
	}
}
