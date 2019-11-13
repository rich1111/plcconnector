package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	plc ".."
)

func call(service int, status int, tag *plc.Tag) {
	switch service {
	case plc.Reset:
		fmt.Println("Reset")
	case plc.ReadTag:
		fmt.Println("Read Tag")
	case plc.WriteTag:
		fmt.Println("Write Tag")
	default:
		fmt.Println("unknown service")
	}
	switch status {
	case plc.Success:
		fmt.Println("Succes")
	case plc.PathSegmentError:
		fmt.Println("PathSegmentError")
	default:
		fmt.Println("unknown status")
	}
	if (service == plc.ReadTag || service == plc.WriteTag) && tag != nil {
		fmt.Println(tag.Name, tag.Count)
		switch tag.Type {
		case plc.TypeBOOL:
			fmt.Println("BOOL type", tag.DataBOOL())
		case plc.TypeSINT:
			fmt.Println("SINT type", tag.DataSINT())
		case plc.TypeINT:
			fmt.Println("INT type", tag.DataINT())
		case plc.TypeDINT:
			fmt.Println("DINT type", tag.DataDINT())
		case plc.TypeREAL:
			fmt.Println("REAL type", tag.DataREAL())
		case plc.TypeDWORD:
			fmt.Println("DWORD type", tag.DataDWORD())
		case plc.TypeLINT:
			fmt.Println("LINT type", tag.DataLINT())
		default:
			fmt.Println("unknown type")
		}
	}
	fmt.Println()
}

func main() {
	signalTrap := make(chan os.Signal, 1)
	signal.Notify(signalTrap, os.Interrupt, syscall.SIGTERM)

	go func() {
		signalType := <-signalTrap
		signal.Stop(signalTrap)
		fmt.Printf("interrupt: %d\n", signalType)
		os.Exit(0)
	}()

	// inicjalizacja
	eds := "test.eds"
	if len(os.Args) >= 2 {
		eds = os.Args[1]
	}
	c, err := plc.Init(eds, true)
	if err != nil {
		fmt.Println(err)
		return
	}

	// nie wyświetlaj dodatkowych informacji
	c.Verbose = true
	c.DumpNetwork = false

	// callback
	// c.Callback(call)

	// strona WWW
	go c.ServeHTTP("0.0.0.0:28080")

	// serwer
	c.Serve("0.0.0.0:44818")
}
