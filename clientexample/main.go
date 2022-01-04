package main

import (
	"fmt"
	"os"

	plc "github.com/podeszfa/plcconnector"
)

func main() {
	addr := "10.1.31.251:44818"
	if len(os.Args) > 1 {
		if os.Args[1] == "d" {
			ids, err := plc.Discover()
			if err != nil {
				fmt.Println(err)
				return
			}
			for _, id := range ids {
				fmt.Printf("%#v\n", id)
			}
		} else {
			addr = os.Args[1]
		}
	}
	c, err := plc.Connect(addr)
	if err != nil {
		fmt.Println(err)
		return
	}

	gaa, err := c.GetAttributesAll(plc.PortClass, 0)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(gaa)

	gal, err := c.GetAttributeList(0x6F, 1, []int{1, 6})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(gal)

	// t, err := c.ReadTag("testSTRUCT", 1)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// fmt.Println(t)
	c.Close()
}
