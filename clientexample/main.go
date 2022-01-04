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
	c, err := plc.Connect(addr, -1)
	if err != nil {
		fmt.Println(err)
		return
	}

	gaa, err := c.GetAttributesAll(plc.IdentityClass, 1)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(gaa)

	gaa, err = c.GetAttributesAll(plc.PortClass, 0)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(gaa)

	gal, err := c.GetAttributeList(0x6F, 1, []int{1, 6})
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(gal)
	}

	gas, err := c.GetAttributeSingle(plc.FileClass, 200, 4)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(gas)

	gas, err = c.GetAttributeSingle(plc.TCPClass, 0, 1)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(gas)

	gas, err = c.GetAttributeSingle(plc.EthernetClass, 0, 1)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(gas)

	gas, err = c.GetAttributeSingle(plc.EthernetClass, 0, 2)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(gas)

	gas, err = c.GetAttributeSingle(plc.DLRClass, 0, 1)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(gas)

	gaa, err = c.GetAttributesAll(plc.DLRClass, 1)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(gaa)

	t, err := c.ReadTag("testSTRUCT", 1)
	if err != nil {
		fmt.Println(err)
		return
	} else {
		fmt.Println(t)
	}
	c.Close()
}
