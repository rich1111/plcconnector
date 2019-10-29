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
		switch tag.Typ {
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
	c := plc.Init(true)

	c.Class[1].Inst[1].Attr[7] = plc.AttrShortString("MongolPLC", "ProductName")

	c.Class[0xF4] = plc.NewClass("Port", 9)
	c.Class[0xF4].Inst[0].Attr[8] = plc.AttrUINT(0, "EntryPort")

	c.Class[0x37] = plc.NewClass("File", 32)
	c.Class[0x37].Inst[0xC8] = plc.NewInstance(11) // EDS
	c.Class[0x37].Inst[0xC8].Attr[4] = plc.AttrStringI("EDS.gz", "FileName")

	// nie wyświetlaj dodatkowych informacji
	c.Verbose = true
	c.DumpNetwork = false

	// callback
	c.Callback(call)

	// strona WWW
	go c.ServeHTTP("0.0.0.0:28080")

	// serwer
	c.Serve("0.0.0.0:44818")
}
