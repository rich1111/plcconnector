// Copyright 2018 Prosap sp. z o.o. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package plcconnector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

func typeToString(t int) string {
	switch t {
	case TypeBOOL:
		return "BOOL"
	case TypeSINT:
		return "SINT"
	case TypeINT:
		return "INT"
	case TypeDINT:
		return "DINT"
	case TypeREAL:
		return "REAL"
	case TypeDWORD:
		return "DWORD"
	case TypeLINT:
		return "LINT"
	default:
		return "UNKNOWN"
	}
}

func asciiCode(x rune) (r string) {
	switch x {
	case 1:
		r = "SOH"
	case 2:
		r = "STX"
	case 3:
		r = "ETX"
	case 4:
		r = "EOT"
	case 5:
		r = "ENQ"
	case 6:
		r = "ACK"
	case 7:
		r = "BEL"
	case 8:
		r = "BS"
	case 9:
		r = "HT"
	case 10:
		r = "LF"
	case 11:
		r = "VT"
	case 12:
		r = "FF"
	case 13:
		r = "CR"
	case 14:
		r = "SO"
	case 15:
		r = "SI"
	case 16:
		r = "DLE"
	case 17:
		r = "DC1"
	case 18:
		r = "DC2"
	case 19:
		r = "DC3"
	case 20:
		r = "DC4"
	case 21:
		r = "NAK"
	case 22:
		r = "SYN"
	case 23:
		r = "ETB"
	case 24:
		r = "CAN"
	case 25:
		r = "EM"
	case 26:
		r = "SUB"
	case 27:
		r = "ESC"
	case 28:
		r = "FS"
	case 29:
		r = "GS"
	case 30:
		r = "RS"
	case 31:
		r = "US"
	default:
		r = "NUL"
	}
	r = "&lt;" + r + "&gt;"
	return
}

func tagsIndexHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	var toSend strings.Builder

	toSend.WriteString("<!DOCTYPE html>\n<html><h3>PLC connector</h3><p>Wersja: 3</p>\n<table><tr><th>Nazwa</th><th>Rozmiar</th><th>Typ</th><th>ASCII</th></tr>\n")

	tMut.RLock()
	arr := make([]string, 0, len(tags))

	for n := range tags {
		arr = append(arr, n)
	}

	sort.Strings(arr)

	for _, n := range arr {
		toSend.WriteString("<tr><td><a href=\"/" + n + "\">" + n + "</a></td><td>" + strconv.Itoa(tags[n].Count) + "</td><td>" + typeToString(tags[n].Typ) + "</td><td>")
		var ascii strings.Builder
		if tags[n].Typ != TypeREAL && tags[n].Typ != TypeBOOL {
			ascii.Grow(tags[n].Count)
			ln := int(typeLen(uint16(tags[n].Typ)))
			for i := 0; i < len(tags[n].data); i += ln {
				tmp := int64(tags[n].data[i])
				for j := 1; j < ln; j++ {
					tmp += int64(tags[n].data[i+j]) << uint(8*j)
				}
				switch tags[n].Typ {
				case TypeSINT:
					tmp = int64(int8(tmp))
				case TypeINT:
					tmp = int64(int16(tmp))
				case TypeDINT:
					tmp = int64(int32(tmp))
				case TypeDWORD:
					tmp = int64(int32(tmp))
				case TypeLINT:
					tmp = int64(int64(tmp))
				}
				if tmp < 256 && tmp >= 0 {
					if tmp < 32 {
						ascii.WriteString(asciiCode(rune(tmp)))
					} else {
						ascii.WriteRune(rune(tmp))
					}
				} else {
					break
				}
			}
			toSend.WriteString(ascii.String())
			toSend.WriteString("</td></tr>\n")
		}
	}
	tMut.RUnlock()

	toSend.WriteString("</table></html>")

	io.WriteString(w, toSend.String())
}

type tagJSON struct {
	Typ   string    `json:"type"`
	Count int       `json:"count"`
	Data  []float64 `json:"data"`
}

func tagToJSON(t *Tag) string {
	var tj tagJSON
	tj.Count = int(t.Count)
	ln := int(typeLen(uint16(t.Typ)))
	for i := 0; i < len(t.data); i += ln {
		tmp := int64(t.data[i])
		for j := 1; j < ln; j++ {
			tmp += int64(t.data[i+j]) << uint(8*j)
		}
		switch t.Typ {
		case TypeBOOL:
			if tmp != 0 {
				tmp = 1
			}
		case TypeSINT:
			tmp = int64(int8(tmp))
		case TypeINT:
			tmp = int64(int16(tmp))
		case TypeDINT:
			tmp = int64(int32(tmp))
		case TypeDWORD:
			tmp = int64(int32(tmp))
		case TypeLINT:
			tmp = int64(int64(tmp))
		}
		if t.Typ == TypeREAL {
			tj.Data = append(tj.Data, float64(math.Float32frombits(uint32(tmp))))
		} else {
			tj.Data = append(tj.Data, float64(tmp))
		}
	}
	tj.Typ = typeToString(t.Typ)

	b, err := json.Marshal(tj)
	if err != nil {
		fmt.Println(err)
		return "{}"
	}
	return string(b)
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		tagsIndexHTML(w, r)
	} else {
		tMut.RLock()
		t, ok := tags[path.Base(r.URL.Path)]
		if ok {
			// if callback != nil {
			// 	callback(ReadTag, Success, &t)
			// }
			str := tagToJSON(t)
			tMut.RUnlock()
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			io.WriteString(w, str)
		} else {
			tMut.RUnlock()
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, "not found")
		}
	}
}

var server *http.Server

// ServeHTTP listens on the TCP network address host.
func ServeHTTP(host string) *http.Server {
	server = &http.Server{Addr: host, Handler: http.HandlerFunc(handler)}
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			fmt.Println("plcconnector ServeHTTP: ", err)
		}
	}()
	return server
}

// CloseHTTP shutdowns the HTTP server
func CloseHTTP() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var err error
	if err = server.Shutdown(ctx); err != nil {
		fmt.Println("plcconnector CloseHTTP: ", err)
	}
	debug("server.Shutdown")
	return err
}
