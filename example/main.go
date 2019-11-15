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

type structTest struct {
	abc uint16
	def float32
}

type structTest2 struct {
	ala  float64
	kot  int32
	pies [8]int8
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
	p.NewTag([]int8{-128, 127, 0, 1}, "testSINT")
	p.NewTag([10]int16{-11, 11, 32767, -32768}, "testINT")
	p.NewTag([]int32{1, -1}, "testDINT")
	p.NewTag([]float32{-0.1, 1.123}, "testREAL")
	p.NewTag([]float64{-0.1, 1.123}, "testLREAL")
	p.NewTag([]int64{-1, 1}, "testLINT")
	p.NewTag([]int8{'H', 'e', 'l', 'l', 'o', '!', 0x00, 0x01, 0x7F, 127}, "testASCII")

	p.NewTag("Ala ma kota", "testSTRING")
	p.NewTag(int32(100), "test1")

	p.NewTag(structTest{abc: 123, def: 1.234}, "testSTRUCT")
	p.NewTag([2]structTest{{abc: 123, def: 1.234}, {abc: 456, def: 7.89}}, "testSTRUCT2")

	p.NewTag([4]structTest2{{ala: 5.5, kot: -111, pies: [8]int8{0, 1, 2, 3, 4, 5, 6, 7}},
		{ala: -5.5, kot: 111, pies: [8]int8{8, 9, 10, 11, 12, 13, 14, 15}}}, "testSTRUCT3")

	err = p.NewTagJSON("test.json", "JSON")
	if err != nil {
		fmt.Println(err)
	}

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
