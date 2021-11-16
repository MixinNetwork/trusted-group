package encoding

const (
	OperationPurposeUnknown       = 0
	OperationPurposeGroupEvent    = 1
	OperationPurposeAddProcess    = 11
	OperationPurposeCreditProcess = 12
)

type Operation struct {
	Purpose  int
	Process  string
	Platform string
	Address  string
	Extra    []byte
}

func DecodeOperation(b []byte) (*Operation, error) {
	panic(0)
}
