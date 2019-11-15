package plcconnector

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
)

func bwrite(buf *bytes.Buffer, data interface{}) {
	err := binary.Write(buf, binary.LittleEndian, data)
	if err != nil {
		fmt.Println(err)
	}
}

func htons(v uint16) uint16 {
	return binary.LittleEndian.Uint16([]byte{byte(v >> 8), byte(v)})
}

func htonl(v uint32) uint32 {
	return binary.LittleEndian.Uint32([]byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)})
}

func getNetIf() (uint32, []byte) {
	ifaces, err := net.Interfaces()
	if err == nil {
		for _, i := range ifaces {
			addrs, err := i.Addrs()
			if err == nil {
				for _, addr := range addrs {
					var ip net.IP
					switch v := addr.(type) {
					case *net.IPNet:
						ip = v.IP
					case *net.IPAddr:
						ip = v.IP
					}
					if !ip.IsLoopback() {
						ipstr := ip.String()
						if !strings.Contains(ipstr, ":") {
							return binary.LittleEndian.Uint32(ip.To4()), i.HardwareAddr
						}
					}
				}
			}
		}
	}
	return 0, nil
}

func getPort(host string) uint16 {
	_, portstr, err := net.SplitHostPort(host)
	if err != nil {
		return 0
	}
	port, err := strconv.Atoi(portstr)
	if err != nil {
		return 0
	}
	return uint16(port)
}

func typeLen(t uint16) uint16 {
	switch t {
	case TypeBOOL:
		return 1
	case TypeSINT:
		return 1
	case TypeINT:
		return 2
	case TypeDINT:
		return 4
	case TypeLINT:
		return 8
	case TypeUSINT:
		return 1
	case TypeUINT:
		return 2
	case TypeUDINT:
		return 4
	case TypeULINT:
		return 8
	case TypeREAL:
		return 4
	case TypeLREAL:
		return 8
	// case TypeSTIME:
	// 	return 1
	// case TypeDATE:
	// 	return 1
	// case TypeTIMEOFDAY:
	// 	return 1
	// case TypeDATETIME:
	// 	return 1
	case TypeSTRING:
		return 1
	case TypeBYTE:
		return 1
	case TypeWORD:
		return 2
	case TypeDWORD:
		return 4
	case TypeLWORD:
		return 8
	case TypeSTRING2:
		return 2
	// case TypeFTIME:
	// 	return 1
	// case TypeLTIME:
	// 	return 1
	// case TypeITIME:
	// 	return 1
	// case TypeSTRINGN:
	// 	return 1
	// case TypeSHORTSTRING:
	// 	return 1
	// case TypeTIME:
	// 	return 1
	// case TypeEPATH:
	// 	return 1
	// case TypeENGUNIT:
	// 	return 1
	// case TypeSTRINGI:
	// 	return 1
	default:
		return 1
	}
}

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
	case TypeLINT:
		return "LINT"
	case TypeUSINT:
		return "USINT"
	case TypeUINT:
		return "UINT"
	case TypeUDINT:
		return "UDINT"
	case TypeULINT:
		return "ULINT"
	case TypeREAL:
		return "REAL"
	case TypeLREAL:
		return "LREAL"
	case TypeSTIME:
		return "STIME"
	case TypeDATE:
		return "DATE"
	case TypeTIMEOFDAY:
		return "TIMEOFDAY"
	case TypeDATETIME:
		return "DATETIME"
	case TypeSTRING:
		return "STRING"
	case TypeBYTE:
		return "BYTE"
	case TypeWORD:
		return "WORD"
	case TypeDWORD:
		return "DWORD"
	case TypeLWORD:
		return "LWORD"
	case TypeSTRING2:
		return "STRING2"
	case TypeFTIME:
		return "FTIME"
	case TypeLTIME:
		return "LTIME"
	case TypeITIME:
		return "ITIME"
	case TypeSTRINGN:
		return "STRINGN"
	case TypeSHORTSTRING:
		return "SHORTSTRING"
	case TypeTIME:
		return "TIME"
	case TypeEPATH:
		return "EPATH"
	case TypeENGUNIT:
		return "ENGUNIT"
	case TypeSTRINGI:
		return "STRINGI"
	default:
		return "UNKNOWN"
	}
}

func asciiCode(x uint8) (r string) {
	switch x {
	case 0:
		r = "NUL"
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
	case 0x7F:
		r = "DEL"
	case 0x80:
		r = "Ç"
	case 0x81:
		r = "ü"
	case 0x82:
		r = "é"
	case 0x83:
		r = "â"
	case 0x84:
		r = "ä"
	case 0x85:
		r = "ů"
	case 0x86:
		r = "ć"
	case 0x87:
		r = "ç"
	case 0x88:
		r = "ł"
	case 0x89:
		r = "ë"
	case 0x8A:
		r = "Ő"
	case 0x8B:
		r = "ő"
	case 0x8C:
		r = "î"
	case 0x8D:
		r = "Ź"
	case 0x8E:
		r = "Ä"
	case 0x8F:
		r = "Ć"
	case 0x90:
		r = "É"
	case 0x91:
		r = "Ĺ"
	case 0x92:
		r = "ĺ"
	case 0x93:
		r = "ô"
	case 0x94:
		r = "ö"
	case 0x95:
		r = "Ľ"
	case 0x96:
		r = "ľ"
	case 0x97:
		r = "Ś"
	case 0x98:
		r = "ś"
	case 0x99:
		r = "Ö"
	case 0x9A:
		r = "Ü"
	case 0x9B:
		r = "Ť"
	case 0x9C:
		r = "ť"
	case 0x9D:
		r = "Ł"
	case 0x9E:
		r = "×"
	case 0x9F:
		r = "č"
	case 0xA0:
		r = "á"
	case 0xA1:
		r = "í"
	case 0xA2:
		r = "ó"
	case 0xA3:
		r = "ú"
	case 0xA4:
		r = "Ą"
	case 0xA5:
		r = "ą"
	case 0xA6:
		r = "Ž"
	case 0xA7:
		r = "ž"
	case 0xA8:
		r = "Ę"
	case 0xA9:
		r = "ę"
	case 0xAA:
		r = "¬"
	case 0xAB:
		r = "ź"
	case 0xAC:
		r = "Č"
	case 0xAD:
		r = "ş"
	case 0xAE:
		r = "«"
	case 0xAF:
		r = "»"
	case 0xB0:
		r = "░"
	case 0xB1:
		r = "▒"
	case 0xB2:
		r = "▓"
	case 0xB3:
		r = "│"
	case 0xB4:
		r = "┤"
	case 0xB5:
		r = "Á"
	case 0xB6:
		r = "Â"
	case 0xB7:
		r = "Ě"
	case 0xB8:
		r = "Ş"
	case 0xB9:
		r = "╣"
	case 0xBA:
		r = "║"
	case 0xBB:
		r = "╗"
	case 0xBC:
		r = "╝"
	case 0xBD:
		r = "Ż"
	case 0xBE:
		r = "ż"
	case 0xBF:
		r = "┐"
	case 0xC0:
		r = "└"
	case 0xC1:
		r = "┴"
	case 0xC2:
		r = "┬"
	case 0xC3:
		r = "├"
	case 0xC4:
		r = "─"
	case 0xC5:
		r = "┼"
	case 0xC6:
		r = "Ă"
	case 0xC7:
		r = "ă"
	case 0xC8:
		r = "╚"
	case 0xC9:
		r = "╔"
	case 0xCA:
		r = "╩"
	case 0xCB:
		r = "╦"
	case 0xCC:
		r = "╠"
	case 0xCD:
		r = "═"
	case 0xCE:
		r = "╬"
	case 0xCF:
		r = "¤"
	case 0xD0:
		r = "đ"
	case 0xD1:
		r = "Đ"
	case 0xD2:
		r = "Ď"
	case 0xD3:
		r = "Ë"
	case 0xD4:
		r = "ď"
	case 0xD5:
		r = "Ň"
	case 0xD6:
		r = "Í"
	case 0xD7:
		r = "Î"
	case 0xD8:
		r = "ě"
	case 0xD9:
		r = "┘"
	case 0xDA:
		r = "┌"
	case 0xDB:
		r = "█"
	case 0xDC:
		r = "▄"
	case 0xDD:
		r = "Ţ"
	case 0xDE:
		r = "Ů"
	case 0xDF:
		r = "▀"
	case 0xE0:
		r = "Ó"
	case 0xE1:
		r = "ß"
	case 0xE2:
		r = "Ô"
	case 0xE3:
		r = "Ń"
	case 0xE4:
		r = "ń"
	case 0xE5:
		r = "ň"
	case 0xE6:
		r = "Š"
	case 0xE7:
		r = "š"
	case 0xE8:
		r = "Ŕ"
	case 0xE9:
		r = "Ú"
	case 0xEA:
		r = "ŕ"
	case 0xEB:
		r = "Ű"
	case 0xEC:
		r = "ý"
	case 0xED:
		r = "Ý"
	case 0xEE:
		r = "ţ"
	case 0xEF:
		r = "´"
	case 0xF0:
		r = "SHY"
	case 0xF1:
		r = "˝"
	case 0xF2:
		r = "˛"
	case 0xF3:
		r = "ˇ"
	case 0xF4:
		r = "˘"
	case 0xF5:
		r = "§"
	case 0xF6:
		r = "÷"
	case 0xF7:
		r = "¸"
	case 0xF8:
		r = "°"
	case 0xF9:
		r = "¨"
	case 0xFA:
		r = "˙"
	case 0xFB:
		r = "ű"
	case 0xFC:
		r = "Ř"
	case 0xFD:
		r = "ř"
	case 0xFE:
		r = "■"
	case 0xFF:
		r = "NBSP"
	default:
		r = string(x)
	}
	return
}
