package routeimporter

import "github.com/open-traffic-generator/snappi/gosnappi"

// route import common structure

// ImportFileType specifies format of the file being imported
type ImportFileType int

const (
	// ImportFileTypeCisco - file in Cisco Route Format
	ImportFileTypeCisco ImportFileType = iota
	// ImportFileTypeJuniper - file in Cisco Route Format
	ImportFileTypeJuniper
)

// RouteType specifies imported route type
type RouteType int

const (
	// RouteTypeAuto - detect route type automatically
	RouteTypeAuto RouteType = iota
	// RouteTypeIpv4 - only IPv4 routes
	RouteTypeIpv4
	// RouteTypeIpv6 - only IPv6 routes
	RouteTypeIpv6
)

// Import configuration specified parameters to control import behavior
type ImportConfig struct {
	NamePrefix        string               // Route name prefix
	RRType            RouteType            // detect route address type
	BestRoutes        bool                 // import best routes only
	RetainNexthop     bool                 // retain next hop
	SequentialProcess bool                 // Process in sequence
	Targetv4Peers     []gosnappi.BgpV4Peer // Target v4 peer that is updated with valid v4 routes
	Targetv6Peers     []gosnappi.BgpV6Peer // Target v6 peer that is updated with valid v6 routes
}

type ImportService interface {
	ImportRoutes(ic ImportConfig, buffer *[]byte) (*[]string, error)
	String() string
}
