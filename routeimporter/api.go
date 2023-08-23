package routeimporter

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
	RRType        RouteType // detect route address type
	BestRoutes    bool      // import best routes only
	RetainNexthop bool      // retain next hop

}

type ImportService interface {
	ImportRoutesBuffer(ic ImportConfig, buffer []byte) error
	String() string
}
