package main

//#include "bridge.h"
import "C"

import (
	"net"

	"strconv"
	"unsafe"

	plc "github.com/podeszfa/plcconnector"
)

func main() {}

var (
	callC   C.intFunc
	p       *plc.PLC
	wasHTTP bool
	wasTCP  bool
)

func call(service int, status int, tag *plc.Tag) {
	f := C.intFunc(callC)

	servC := C.int(service)
	statC := C.int(status)
	var nameC *C.char
	nameC = nil
	typeC := C.int(0)
	countC := C.int(0)
	var dataC unsafe.Pointer
	dataC = nil

	if tag != nil {
		nameC = C.CString(tag.Name)
		typeC = C.int(tag.Type)
		countC = C.int(tag.Dim[0])
		dataC = C.CBytes(tag.DataBytes())
	}

	C.bridge_int_func(f, servC, statC, nameC, typeC, countC, dataC)
}

//export plcconnector_init
func plcconnector_init() {
	p, _ = plc.Init("")
}

//export plcconnector_set_verbose
func plcconnector_set_verbose(on C.int) {
	if on == 0 {
		p.Verbose = false
	} else {
		p.Verbose = true
	}
}

//export plcconnector_callback
func plcconnector_callback(f C.intFunc) {
	callC = f
	p.Callback(call)
}

//export plcconnector_serve
func plcconnector_serve(host *C.char, port C.int) {
	wasTCP = true
	url := net.JoinHostPort(C.GoString(host), strconv.Itoa(int(port)))
	go p.Serve(url)
}

//export plcconnector_serve_http
func plcconnector_serve_http(host *C.char, port C.int) {
	wasHTTP = true
	url := net.JoinHostPort(C.GoString(host), strconv.Itoa(int(port)))
	p.ServeHTTP(url)
}

//export plcconnector_add_tag
func plcconnector_add_tag(name *C.char, typ, count C.int) {
	t := plc.Tag{Name: C.GoString(name), Type: int(typ), Dim: [3]int{int(count), 0, 0}}
	p.AddTag(t)
}

//export plcconnector_update_tag
func plcconnector_update_tag(name *C.char, offset C.int, data *C.char, sizeOf C.int) C.int {
	r := p.UpdateTag(C.GoString(name), int(offset), C.GoBytes(unsafe.Pointer(data), sizeOf))
	if r {
		return 1
	}
	return 0
}

//export plcconnector_close
func plcconnector_close() {
	if wasHTTP {
		p.CloseHTTP()
	}
	if wasTCP {
		p.Close()
	}
}
