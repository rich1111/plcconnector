package plcconnector

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
)

func bread(rd io.Reader, data interface{}) error {
	err := binary.Read(rd, binary.LittleEndian, data)
	if err != nil {
		fmt.Println(err)
	}
	return err
}

func bwrite(buf io.Writer, data interface{}) {
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
		if t&0xFFFF0000 == TypeStructHead {
			return "STRUCT"
		}
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

var crc16Table = [...]uint16{
	0x0000, 0xC0C1, 0xC181, 0x0140, 0xC301, 0x03C0, 0x0280, 0xC241,
	0xC601, 0x06C0, 0x0780, 0xC741, 0x0500, 0xC5C1, 0xC481, 0x0440,
	0xCC01, 0x0CC0, 0x0D80, 0xCD41, 0x0F00, 0xCFC1, 0xCE81, 0x0E40,
	0x0A00, 0xCAC1, 0xCB81, 0x0B40, 0xC901, 0x09C0, 0x0880, 0xC841,
	0xD801, 0x18C0, 0x1980, 0xD941, 0x1B00, 0xDBC1, 0xDA81, 0x1A40,
	0x1E00, 0xDEC1, 0xDF81, 0x1F40, 0xDD01, 0x1DC0, 0x1C80, 0xDC41,
	0x1400, 0xD4C1, 0xD581, 0x1540, 0xD701, 0x17C0, 0x1680, 0xD641,
	0xD201, 0x12C0, 0x1380, 0xD341, 0x1100, 0xD1C1, 0xD081, 0x1040,
	0xF001, 0x30C0, 0x3180, 0xF141, 0x3300, 0xF3C1, 0xF281, 0x3240,
	0x3600, 0xF6C1, 0xF781, 0x3740, 0xF501, 0x35C0, 0x3480, 0xF441,
	0x3C00, 0xFCC1, 0xFD81, 0x3D40, 0xFF01, 0x3FC0, 0x3E80, 0xFE41,
	0xFA01, 0x3AC0, 0x3B80, 0xFB41, 0x3900, 0xF9C1, 0xF881, 0x3840,
	0x2800, 0xE8C1, 0xE981, 0x2940, 0xEB01, 0x2BC0, 0x2A80, 0xEA41,
	0xEE01, 0x2EC0, 0x2F80, 0xEF41, 0x2D00, 0xEDC1, 0xEC81, 0x2C40,
	0xE401, 0x24C0, 0x2580, 0xE541, 0x2700, 0xE7C1, 0xE681, 0x2640,
	0x2200, 0xE2C1, 0xE381, 0x2340, 0xE101, 0x21C0, 0x2080, 0xE041,
	0xA001, 0x60C0, 0x6180, 0xA141, 0x6300, 0xA3C1, 0xA281, 0x6240,
	0x6600, 0xA6C1, 0xA781, 0x6740, 0xA501, 0x65C0, 0x6480, 0xA441,
	0x6C00, 0xACC1, 0xAD81, 0x6D40, 0xAF01, 0x6FC0, 0x6E80, 0xAE41,
	0xAA01, 0x6AC0, 0x6B80, 0xAB41, 0x6900, 0xA9C1, 0xA881, 0x6840,
	0x7800, 0xB8C1, 0xB981, 0x7940, 0xBB01, 0x7BC0, 0x7A80, 0xBA41,
	0xBE01, 0x7EC0, 0x7F80, 0xBF41, 0x7D00, 0xBDC1, 0xBC81, 0x7C40,
	0xB401, 0x74C0, 0x7580, 0xB541, 0x7700, 0xB7C1, 0xB681, 0x7640,
	0x7200, 0xB2C1, 0xB381, 0x7340, 0xB101, 0x71C0, 0x7080, 0xB041,
	0x5000, 0x90C1, 0x9181, 0x5140, 0x9301, 0x53C0, 0x5280, 0x9241,
	0x9601, 0x56C0, 0x5780, 0x9741, 0x5500, 0x95C1, 0x9481, 0x5440,
	0x9C01, 0x5CC0, 0x5D80, 0x9D41, 0x5F00, 0x9FC1, 0x9E81, 0x5E40,
	0x5A00, 0x9AC1, 0x9B81, 0x5B40, 0x9901, 0x59C0, 0x5880, 0x9841,
	0x8801, 0x48C0, 0x4980, 0x8941, 0x4B00, 0x8BC1, 0x8A81, 0x4A40,
	0x4E00, 0x8EC1, 0x8F81, 0x4F40, 0x8D01, 0x4DC0, 0x4C80, 0x8C41,
	0x4400, 0x84C1, 0x8581, 0x4540, 0x8701, 0x47C0, 0x4680, 0x8641,
	0x8201, 0x42C0, 0x4380, 0x8341, 0x4100, 0x81C1, 0x8081, 0x4040,
}

func crc16(buffer []byte) uint16 {
	crc := uint16(0)
	tmp := uint8(0)

	for i := 0; i < len(buffer); i++ {
		tmp = uint8(uint16(buffer[i]) ^ crc)
		crc >>= 8
		crc ^= crc16Table[tmp]
	}
	return crc
}
