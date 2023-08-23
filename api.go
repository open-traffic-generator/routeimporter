package routeimporter

// route import common structure

// ImportFileType specifies format of import file type and store it
type ImportFileType int

const (
	// ImportFileTypeCisco TBD
	ImportFileTypeCisco ImportFileType = iota
	// ImportFileTypeJuniper TBD
	ImportFileTypeJuniper
)

// Connected Device interface type
type RouteType int

const (
	RouteTypeAuto RouteType = iota
	RouteTypeIpv4
	RouteTypeIpv6
)

// set parameters like next hop, best routes etc as parameter
type ImportConfig struct {
	BestRoutes   bool      // import best routes only
	LocalNexthop bool      // retain next hop
	RRType       RouteType // detect route address type
}

type ImportService interface {
	ImportRoutesBuffer(ic ImportConfig, buffer []byte) error
	String() string
}
