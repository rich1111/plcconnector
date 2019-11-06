package plcconnector

// Service
const (
	GetAttrAll      = 0x01
	GetAttrList     = 0x03
	Reset           = 0x05
	GetAttr         = 0x0E
	ForwardOpen     = 0x54
	ForwardClose    = 0x4E
	ReadTag         = 0x4C
	WriteTag        = 0x4D
	InititateUpload = 0x4B
	UploadTransfer  = 0x4F
	GetInstAttrList = 0x55
)

// Data types
const (
	TypeBOOL  = 0xC1 // 1 byte
	TypeSINT  = 0xC2 // 1 byte
	TypeINT   = 0xC3 // 2 bytes
	TypeDINT  = 0xC4 // 4 bytes
	TypeREAL  = 0xCA // 4 bytes
	TypeDWORD = 0xD3 // 4 bytes
	TypeLINT  = 0xC5 // 8 bytes

	TypeUSINT = 0xC6
	TypeUINT  = 0xC7
	TypeUDINT = 0xC8

	TypeString      = 0xD0
	TypeShortString = 0xDA
	TypeStringI     = 0x04
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
	itNullAddress = 0x00
	itUnconnData  = 0xb2
	itConnAddress = 0xa1
	itConnData    = 0xb1

	itListIdentity = 0x0C
	itListService  = 0x100
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
