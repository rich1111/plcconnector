package main

import (
	"fmt"
	"math"
	"os"
	"os/signal"
	"syscall"

	plc "github.com/rich1111/plcconnector"
)


const assemblyInInstance = 0x65
const assemblyOutInstance = 0x66


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
		fmt.Println(tag.Name, tag.Dim)
		switch tag.BasicType() {
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
	eds := ""
	if len(os.Args) >= 2 {
		eds = os.Args[1]
	}
	p, err := plc.Init(eds)
	if err != nil {
		fmt.Println(err)
		return
	}

	if len(os.Args) >= 3 {
		err = p.ImportJSON(os.Args[2])
		if err != nil {
			fmt.Println(err)
			return
		}
		if len(os.Args) >= 4 {
			err = p.ImportMemoryJSON(os.Args[3])
			if err != nil {
				fmt.Println(err)
				return
			}
		}
	} else {
		p.NewTag(bool(false), "testBOOL")
		p.NewTag([4]bool{false, true}, "testBOOL4")

		p.NewTag([]int8{1, -128, 127}, "testSINT")
		p.NewTag([]uint8{1, 0xFF, 0}, "testUSINT")

		p.NewTag([]int16{1, -32768, 32767}, "testINT")
		p.NewTag([]uint16{1, 0xFFFF, 0}, "testUINT")

		p.NewTag([]int32{1, -2147483648, 2147483647}, "testDINT")
		p.NewTag([]uint32{1, 0xFFFFFFFF, 0}, "testUDINT")

		p.NewTag([]int64{1, -9223372036854775808, 9223372036854775807}, "testLINT")
		p.NewTag([]uint64{1, 0xFFFFFFFFFFFFFFFF, 0}, "testULINT")

		p.NewTag([]float32{-0.1, 1.123, 0, math.Float32frombits(0x80000000), float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1))}, "testREAL")
		p.NewTag([]float64{-0.1, 1.123, 0, math.Float64frombits(0x8000000000000000), math.NaN(), math.Inf(1), math.Inf(-1)}, "testLREAL")

		p.NewTag([]int8{'H', 'e', 'l', 'l', 'o', '!', 0x00, 0x01, 0x7F, 127}, "testASCII")

		p.NewTag("Ala ma kota", "testSTRING")
		p.NewTag(int32(100), "test1")

		p.NewTag(structTest{abc: 123, def: 1.234}, "testSTRUCT")
		p.NewTag([2]structTest{{abc: 123, def: 1.234}, {abc: 456, def: 7.89}}, "testSTRUCT2")

		p.NewTag([4]structTest2{
			{ala: 5.5, kot: -111, pies: [8]int8{0, 1, 2, 3, 4, 5, 6, 7}},
			{ala: -5.5, kot: 111, pies: [8]int8{8, 9, 10, 11, 12, 13, 14, 15}},
		}, "testSTRUCT3")

		p.NewUDT("DATATYPE POSITION DINT x; DINT y; END_DATATYPE")
		p.NewUDT("DATATYPE HMM (FamilyType := NoFamily) POSITION sprites[8]; LINT money; END_DATATYPE")
		p.NewUDT("DATATYPE POSITION3D (FamilyType := NoFamily) DINT x; DINT y; DINT z; END_DATATYPE")
		p.NewUDT("DATATYPE MHH (FamilyType := NoFamily) POSITION3D objects[2]; SINT lives; REAL temp; LREAL temp2[3]; END_DATATYPE")

		p.NewUDT("DATATYPE BOOLS (FamilyType := NoFamily) BOOL In; BOOL Out; END_DATATYPE")
		p.NewUDT("DATATYPE STRINSTR (FamilyType := NoFamily) INT int; BOOLS struct[2]; END_DATATYPE")
		p.NewUDT("DATATYPE ASCIISTRING82 DINT LEN; SINT DATA[82]; END_DATATYPE")

		p.CreateTag("ASCIISTRING82", "testASCIISTRING")

		p.CreateTag("POSITION", "pos1")
		p.CreateTag("HMM", "hmm1")
		p.CreateTag("MHH", "mhh1")
		p.CreateTag("BOOLS", "testBOOLS")
		p.CreateTag("STRINSTR", "testSIS")

		p.CreateTag("INT[4,4]", "array2D")
		p.CreateTag("INT[4,4,4]", "array3D")

		p.CreateTag("INT[2,4]", "array2D_2")
		p.CreateTag("INT[2,4,8]", "array3D_2")

		p.CreateTag("INT[4,2]", "array2D_3")
		p.CreateTag("INT[8,2,4]", "array3D_3")



		// ADD Assembly Class
		p.CreateDefaultAssemblyClass(assemblyInInstance, assemblyOutInstance)

		p.NewUDT("DATATYPE IN8DI8DO USINT diStatus; UDINT diCounterValue[8]; USINT doStatus; END_DATATYPE")
		p.NewUDT("DATATYPE OUT8DI8DO USINT doStatus; END_DATATYPE")

		p.CreateInOutTagForAssemblyClass("IN8DI8DO", "testIN8DI8DO", assemblyInInstance, false, assemblyClassInputGetter, nil)
		p.SetSizeTagForAssemblyClass(assemblyInInstance, 34)

		p.CreateInOutTagForAssemblyClass("OUT8DI8DO", "testOUT8DI8DO", assemblyOutInstance, true, nil,
			func (data []uint8) uint8 {
				fmt.Println(data)
				p.GetClassInstance(plc.AssemblyClass, assemblyOutInstance).SetAttrUSINT(3, data[0])
				return 0
		})
		p.SetSizeTagForAssemblyClass(assemblyOutInstance, 1)
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


func assemblyClassInputGetter() []uint8 {
	return []uint8{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34}
}
