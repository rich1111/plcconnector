package plcconnector

// Service
const (
	// Common
	GetAttrAll  = 0x01
	GetAttrList = 0x03
	Reset       = 0x05
	GetAttr     = 0x0E
	// Class Specific
	InititateUpload = 0x4B
	ReadTag         = 0x4C
	WriteTag        = 0x4D
	ForwardClose    = 0x4E
	UploadTransfer  = 0x4F
	ForwardOpen     = 0x54
	GetInstAttrList = 0x55
)

// Data types
const (
	TypeBOOL        = 0xC1 // 1
	TypeSINT        = 0xC2 // 1
	TypeINT         = 0xC3 // 2
	TypeDINT        = 0xC4 // 4
	TypeLINT        = 0xC5 // 8
	TypeUSINT       = 0xC6 // 1
	TypeUINT        = 0xC7 // 2
	TypeUDINT       = 0xC8 // 4
	TypeULINT       = 0xC9 // 8
	TypeREAL        = 0xCA // 4
	TypeLREAL       = 0xCB // 8
	TypeSTIME       = 0xCC // synchronous time
	TypeDATE        = 0x0CD
	TypeTIMEOFDAY   = 0xCE
	TypeDATETIME    = 0xCF
	TypeSTRING      = 0xD0 // 1
	TypeBYTE        = 0xD1 // 1
	TypeWORD        = 0xD2 // 2
	TypeDWORD       = 0xD3 // 4
	TypeLWORD       = 0xD4 // 8
	TypeSTRING2     = 0xD5 // 2
	TypeFTIME       = 0xD6 // duration high resolution
	TypeLTIME       = 0xD7 // duration long
	TypeITIME       = 0xD8 // duration short
	TypeSTRINGN     = 0xD9 // n
	TypeSHORTSTRING = 0xDA
	TypeTIME        = 0xDB // duration miliseconds
	TypeEPATH       = 0xDC
	TypeENGUNIT     = 0xDD // engineering units
	TypeSTRINGI     = 0xDE

	TypeArray1D = 0x2000
	TypeArray2D = 0x4000
	TypeArray3D = 0x6000
	TypeStruct  = 0x8000
)

// Status codes
const (
	Success          = 0x00
	PathSegmentError = 0x04
	PathUnknown      = 0x05
	ServNotSup       = 0x08
	AttrListError    = 0x0A
	AttrNotSup       = 0x14
	InvalidPar       = 0x20
)

// EIP Error Codes
const (
	eipSuccess                = 0x00
	eipInvalid                = 0x01
	eipNoMemory               = 0x02
	eipIncorrectData          = 0x03
	eipInvalidSessionHandle   = 0x64
	eipInvalidLength          = 0x65
	eipInvalidProtocolVersion = 0x69
)

// EIP Encapsulationn Commands
const (
	ecNOP               = 0x00
	ecListServices      = 0x04
	ecListIdentity      = 0x63
	ecListInterfaces    = 0x64
	ecRegisterSession   = 0x65
	ecUnRegisterSession = 0x66
	ecSendRRData        = 0x6f
	ecSendUnitData      = 0x70
	ecIndicateStatus    = 0x72
	ecCancel            = 0x73
)

// Item Type Codes
const (
	itNullAddress  = 0x0000
	itListIdentity = 0x000C
	itConnAddress  = 0x00A1
	itConnData     = 0x00B1
	itUnconnData   = 0x00B2
	itListService  = 0x0100
	itSockAddrOT   = 0x8000
	itSockAddrTO   = 0x8001
	itSeqAddress   = 0x8002
)

// ListService Communications Capability Flags
const (
	lscfTCP = 32
	lscfUDP = 256
)

type encapsulationHeader struct {
	Command       uint16
	Length        uint16
	SessionHandle uint32
	Status        uint32
	SenderContext uint64
	Options       uint32
}

type registerSessionData struct {
	ProtocolVersion uint16
	OptionFlags     uint16
}

type listServicesData struct {
	ProtocolVersion uint16
	CapabilityFlags uint16
	NameOfService   [16]int8
}

type listIdentityData struct {
	ProtocolVersion uint16
	SocketFamily    uint16
	SocketPort      uint16
	SocketAddr      uint32
	SocketZero      [8]uint8
}

type sendData struct {
	InterfaceHandle uint32
	Timeout         uint16
	ItemCount       uint16
}

type itemType struct {
	Type   uint16
	Length uint16
}

type protocolData struct {
	Service  uint8
	PathSize uint8
}

type forwardOpenData struct {
	TimeOut                uint16
	OTConnectionID         uint32
	TOConnectionID         uint32
	ConnSerialNumber       uint16
	VendorID               uint16
	OriginatorSerialNumber uint32
	ConnTimeoutMult        uint8
	_                      [3]uint8
	OTRPI                  uint32
	OTConnPar              uint16
	TORPI                  uint32
	TOConnPar              uint16
	TransportType          uint8
	ConnPathSize           uint8
}

type forwardCloseData struct {
	TimeOut                uint16
	ConnSerialNumber       uint16
	VendorID               uint16
	OriginatorSerialNumber uint32
	ConnPathSize           uint8
	_                      uint8
}

type forwardOpenResponse struct {
	OTConnectionID         uint32
	TOConnectionID         uint32
	ConnSerialNumber       uint16
	VendorID               uint16
	OriginatorSerialNumber uint32
	OTAPI                  uint32
	TOAPI                  uint32
	AppReplySize           uint8
	_                      uint8
}

type forwardCloseResponse struct {
	ConnSerialNumber       uint16
	VendorID               uint16
	OriginatorSerialNumber uint32
	AppReplySize           uint8
	_                      uint8
}

type initUploadResponse struct {
	FileSize     uint32
	TransferSize uint8
}

const (
	tptFirst     = 0
	tptMiddle    = 1
	tptLast      = 2
	tptAbort     = 3
	tptFirstLast = 4
)

type uploadTransferResponse struct {
	TransferNumber    uint8
	TranferPacketType uint8
}

type response struct {
	Service       uint8
	_             uint8
	Status        uint8
	AddStatusSize uint8
}
