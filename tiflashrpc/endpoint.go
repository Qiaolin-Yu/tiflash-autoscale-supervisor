package tiflashrpc

// EndpointType represents the type of a remote endpoint..
type EndpointType uint8

// EndpointType type enums.
const (
	TiKV EndpointType = iota
	TiFlash
	TiDB
)

// Name returns the name of endpoint type.
func (t EndpointType) Name() string {
	switch t {
	case TiKV:
		return "tikv"
	case TiFlash:
		return "tiflash"
	case TiDB:
		return "tidb"
	}
	return "unspecified"
}
