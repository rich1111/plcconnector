// Service
enum {
	Reset        = 0x05,
	ForwardOpen  = 0x54,
	ForwardClose = 0x4e,
	ReadTag      = 0x4c,
	WriteTag     = 0x4d
};

// Data types
enum {
	TypeBOOL  = 0xc1, // 1 byte
	TypeSINT  = 0xc2, // 1 byte
	TypeINT   = 0xc3, // 2 bytes
	TypeDINT  = 0xc4, // 4 bytes
	TypeREAL  = 0xca, // 4 bytes
	TypeDWORD = 0xd3, // 4 bytes
	TypeLINT  = 0xc5 // 8 bytes
};

// Status codes
enum {
	Success          = 0x00,
	PathSegmentError = 0x04
};

typedef void (*intFunc) (int, int, char *, int, int, void *);

void bridge_int_func(intFunc f, int service, int status, char *name, int type, int count, void *data);
