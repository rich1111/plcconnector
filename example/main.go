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

type test struct {
	abc uint16
	def float32
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
	p, err := plc.Init(eds)
	if err != nil {
		fmt.Println(err)
		return
	}

	p.NewTag([4]bool{false, true}, "testBOOL")
	p.NewTag([4]int8{-128, 127, 0, 1}, "testSINT")
	p.NewTag([10]int16{-11, 11, 32767, -32768}, "testINT")
	p.NewTag([2]int32{1, -1}, "testDINT")
	p.NewTag([2]float32{-0.1, 1.123}, "testREAL")
	p.NewTag([2]float64{-0.1, 1.123}, "testLREAL")
	p.NewTag([2]int64{-1, 1}, "testLINT")
	p.NewTag([10]int8{'H', 'e', 'l', 'l', 'o', '!', 0x00, 0x01, 0x7F, 127}, "testASCII")

	p.NewTag([20]int8{0x53, 0x54, 0x52, 0x55, 0x43, 0x54, 0x5F, 0x42, 0x3B, 0x6E, 0x45, 0x42, 0x45, 0x43, 0x45, 0x41, 0x48, 0x41, 0x00}, "1")
	p.NewTag([20]int8{0x5A, 0x5A, 0x5A, 0x5A, 0x5A, 0x5A, 0x5A, 0x5A, 0x5A, 0x5A, 0x53, 0x54, 0x52, 0x55, 0x43, 0x54, 0x5F, 0x42, 0x30, 0x00}, "2")

	p.NewTag("Ala ma kota", "testSTRING")
	p.NewTag(int32(100), "test1")

	// p.NewTag(test{abc: 123, def: 1.234}, "testSTRUCT")

	// nie wyświetlaj dodatkowych informacji
	p.Verbose = true
	p.DumpNetwork = false

	// callback
	// p.Callback(call)

	// strona WWW
	go p.ServeHTTP("0.0.0.0:28080")

	// serwer
	p.Serve("0.0.0.0:44818")
}
