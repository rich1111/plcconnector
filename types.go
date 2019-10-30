package plcconnector

// Service
const (
	GetAttrAll      = 0x01
	Reset           = 0x05
	GetAttr         = 0x0E
	ForwardOpen     = 0x54
	ForwardClose    = 0x4E
	ReadTag         = 0x4C
	WriteTag        = 0x4D
	InititateUpload = 0x4B
	UploadTransfer  = 0x4F
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

	TypeShortString = 0xDA
	TypeStringI     = 0x04
)

// Status codes
const (
	Success          = 0x00
	PathSegmentError = 0x04
)

const (
	nop               = 0x00
	listServices      = 0x04
	listIdentity      = 0x63
	listInterfaces    = 0x64
	registerSession   = 0x65
	sendRRData        = 0x6f
	sendUnitData      = 0x70
	unregisterSession = 0x66

	nullAddressItem = 0x00
	unconnDataItem  = 0xb2
	connAddressItem = 0xa1
	connDataItem    = 0xb1

	ansiExtended = 0x91

	capabilityFlagsCipTCP          = 32
	capabilityFlagsCipUDPClass0or1 = 256

	cipItemIDListServiceResponse = 0x100

	pathClass = 0x20
	pathInst  = 0x24
	pathAttr  = 0x28
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
	ProtocolVersion   uint16
	SocketFamily      uint16
	SocketPort        uint16
	SocketAddr        uint32
	SocketZero        [8]uint8
	VendorID          uint16
	DeviceType        uint16
	ProductCode       uint16
	Revision          [2]uint8
	Status            uint16
	SerialNumber      uint32
	ProductNameLength uint8
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
