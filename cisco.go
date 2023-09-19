package routeimporter

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/open-traffic-generator/snappi/gosnappi"
	"github.com/rs/zerolog/log"
)

const (
	CISCO_HEADER_CHECK_STRING = "   Network" // Starts with 3 spaces.
	CISCO_HEADER_NETWORK      = "Network"
	CISCO_HEADER_NEXT_HOP     = "Next Hop"
	CISCO_HEADER_METRIC       = "Metric"
	CISCO_HEADER_LOC_PRF      = "LocPrf"
	CISCO_HEADER_WEIGHT       = "Weight"
	CISCO_HEADER_PATH         = "Path"

	CISCO_VALID_ROUTE = '*'
	CISCO_BEST_ROUTE  = '>'

	SPACE_CHAR = ' '

	CISCO_VALID_ROUTE_OFFSET = 0
	CISCO_BEST_ROUTE_OFFSET  = 1
)

type rrEntry struct {
	Prefix string
	Row    int
	RRv4   gosnappi.BgpV4RouteRange
	RRv6   gosnappi.BgpV6RouteRange
	Err    *error
}

type CiscoImporter struct {
	id uint64

	//
	POS_CISCO_HEADER_NETWORK  int
	POS_CISCO_HEADER_NEXT_HOP int
	POS_CISCO_HEADER_METRIC   int
	POS_CISCO_HEADER_LOC_PRF  int
	POS_CISCO_HEADER_WEIGHT   int
	POS_CISCO_HEADER_PATH     int

	validRoutes int
	startTask   time.Time
	lines       []string
	PeerV4      gosnappi.BgpV4Peer
}

// String returns the id of the client.
func (imp *CiscoImporter) String() string {
	return fmt.Sprintf("Cisco Route Importer, session id: %8d, validRoutes:%d",
		imp.id, imp.validRoutes)
}

func (imp *CiscoImporter) ImportRoutes(ic ImportConfig, buffer *[]byte) (*[]string, error) {
	if buffer == nil || len(*buffer) == 0 {
		return nil, fmt.Errorf("cannot import - empty route buffer")
	}

	imp.startTask = time.Now()
	imp.lines = strings.Split(string(*buffer), "\n")
	var next int = 0
	var err error = nil
	if next, err = imp.TryParseHeader(); err != nil {
		return nil, fmt.Errorf("cannot import, header not found - %v", err.Error())
	}
	log.Info().Int64("milisecs", time.Since(imp.startTask).Milliseconds()).Msg("Header Parsing")
	if len(ic.Targetv4Peers) > 0 {
		if len(ic.Targetv4Peers) > 1 {
			// To be handled in future
			return nil, fmt.Errorf("multiple target v4 peers currently not supported")
		}
		imp.PeerV4 = ic.Targetv4Peers[0]
	} else {
		return nil, fmt.Errorf("cannot import, no target v4 peers found")
	}

	imp.startTask = time.Now()
	var prefix string
	rrEntryList := []rrEntry{}
	for index := next; index < len(imp.lines); index++ {
		if strings.ContainsAny(imp.lines[index], "\t") {
			return nil, fmt.Errorf("Invalid format - contains tab character (line %v)", index+1)
		}
		if len(imp.lines[index]) == 0 || isSkippableLine(&imp.lines[index]) {
			continue
		}
		pos := imp.POS_CISCO_HEADER_NETWORK
		if imp.lines[index][pos] != SPACE_CHAR {
			offset := strings.Index(imp.lines[index][pos:], " ")
			if offset == -1 {
				prefix = imp.lines[index][pos:]
			} else {
				prefix = imp.lines[index][pos:(offset + pos)]
			}
		}
		if !isValidRoute(&imp.lines[index]) {
			// Invalid route - likely extended from last line
			continue
		}
		if ic.BestRoutes && imp.lines[index][CISCO_BEST_ROUTE_OFFSET] != CISCO_BEST_ROUTE {
			continue
		}
		rrEntryList = append(rrEntryList, rrEntry{Prefix: prefix, Row: index})
	}
	if ic.SequentialProcess {
		for i, _ := range rrEntryList {
			imp.ProcessRR(&rrEntryList[i], &ic)
		}
	} else {
		var wg sync.WaitGroup
		for i, _ := range rrEntryList {
			wg.Add(1)
			go func(entry *rrEntry) {
				defer wg.Done()
				imp.ProcessRR(entry, &ic)
			}(&rrEntryList[i])
		}
		wg.Wait()
	}
	log.Info().Int64("milisecs", time.Since(imp.startTask).Milliseconds()).Msg("Config update")

	imp.startTask = time.Now()
	route_names := []string{}
	for _, rre := range rrEntryList {
		if rre.RRv4 != nil {
			imp.PeerV4.V4Routes().Append(rre.RRv4)
			name := fmt.Sprintf("%s-%d", ic.SessionName, rre.Row)
			route_names = append(route_names, name)
			imp.validRoutes++
		} else {
			fmt.Printf("No result for row %d\n", rre.Row+1)
		}
	}

	return &route_names, nil
}

func (imp *CiscoImporter) TryParseHeader() (int, error) {
	for index, line := range imp.lines {
		if strings.HasPrefix(line, CISCO_HEADER_CHECK_STRING) {
			if strings.ContainsAny(line, "\t") {
				return -1, fmt.Errorf("invalid format - header contains tab character")
			}
			if err := imp.GetHeaderPositions(line); err != nil {
				log.Info().Int64("milisecs", time.Since(imp.startTask).Milliseconds()).Msgf("%v", err)
			} else {
				return index + 1, nil
			}
		}
	}

	return -1, fmt.Errorf("invalid format - failed to locate header")
}

func (imp *CiscoImporter) GetHeaderPositions(line string) error {
	var pos int

	pos = strings.Index(line[pos:], CISCO_HEADER_NETWORK)
	if pos == -1 {
		return fmt.Errorf("Invalid header format - missing Network")
	}
	imp.POS_CISCO_HEADER_NETWORK = pos

	offset := pos + len(CISCO_HEADER_NETWORK)
	pos = strings.Index(line[offset:], CISCO_HEADER_NEXT_HOP)
	if pos == -1 {
		return fmt.Errorf("Invalid header format - missing Next Hop")
	}
	pos = pos + offset
	imp.POS_CISCO_HEADER_NEXT_HOP = pos

	offset = pos + len(CISCO_HEADER_NEXT_HOP)
	pos = strings.Index(line[offset:], CISCO_HEADER_METRIC)
	if pos == -1 {
		return fmt.Errorf("Invalid header format - missing Metric")
	}
	pos = pos + offset
	imp.POS_CISCO_HEADER_METRIC = pos

	offset = pos + len(CISCO_HEADER_METRIC)
	pos = strings.Index(line[offset:], CISCO_HEADER_LOC_PRF)
	if pos == -1 {
		return fmt.Errorf("Invalid header format - missing LocPrf")
	}
	pos = pos + offset
	imp.POS_CISCO_HEADER_LOC_PRF = pos

	offset = pos + len(CISCO_HEADER_LOC_PRF)
	pos = strings.Index(line[offset:], CISCO_HEADER_WEIGHT)
	if pos == -1 {
		return fmt.Errorf("Invalid header format - missing Weight")
	}
	pos = pos + offset
	imp.POS_CISCO_HEADER_WEIGHT = pos

	offset = pos + len(CISCO_HEADER_WEIGHT)
	pos = strings.Index(line[offset:], CISCO_HEADER_PATH)
	if pos == -1 {
		return fmt.Errorf("Invalid header format - missing Path")
	}
	pos = pos + offset
	imp.POS_CISCO_HEADER_PATH = pos

	return nil
}

func (imp *CiscoImporter) ParseNext(pos int, next int, row *int) string {
	line := imp.lines[*row]
	for len(line) <= pos {
		if len(imp.lines) <= *row || isValidRoute(&imp.lines[*row+1]) {
			return ""
		}
		*row = *row + 1
		line = imp.lines[*row]
	}
	if len(line) <= next {
		return strings.TrimSpace(line[pos:])
	} else {
		return strings.TrimSpace(line[pos:next])
	}
}

// func (imp *CiscoImporter) ProcessRR(rri RRInfo, ic *ImportConfig) error {
func (imp *CiscoImporter) ProcessRR(rre *rrEntry, ic *ImportConfig) {
	var ip net.IP
	var mask int
	var err error = nil
	var rrV4 gosnappi.BgpV4RouteRange = nil
	network := rre.Prefix

	nextHop := ""
	if ic.RetainNexthop {
		if nextHop = imp.ParseNext(imp.POS_CISCO_HEADER_NEXT_HOP, imp.POS_CISCO_HEADER_METRIC, &rre.Row); nextHop == "" {
			pErr := fmt.Errorf("no nexthop found (line %d)", rre.Row+1)
			log.Info().Msgf(pErr.Error())
			rre.Err = &pErr
			return
		}
	}
	metric := imp.ParseNext(imp.POS_CISCO_HEADER_METRIC, imp.POS_CISCO_HEADER_LOC_PRF, &rre.Row)
	locPrf := imp.ParseNext(imp.POS_CISCO_HEADER_LOC_PRF, imp.POS_CISCO_HEADER_WEIGHT, &rre.Row)
	// weight := imp.ParseNext(imp.POS_CISCO_HEADER_WEIGHT, imp.POS_CISCO_HEADER_PATH, &rre.Row)
	path := imp.ParseNext(imp.POS_CISCO_HEADER_PATH, len(imp.lines[rre.Row]), &rre.Row)

	if ip, mask, err = ParseNetworkAddress(network); err != nil {
		pErr := fmt.Errorf("Row: %d, Network Address parsing error:%s", rre.Row+1, err.Error())
		log.Info().Msgf(pErr.Error())
		rre.Err = &pErr
		return
	}
	name := fmt.Sprintf("%s-%d", ic.SessionName, rre.Row+1)
	if ip.To4() != nil && (ic.RRType == RouteTypeIpv4 || ic.RRType == RouteTypeAuto) {
		rrV4 = gosnappi.NewBgpV4RouteRange()
		rrV4.SetName(name)
		rrV4.Addresses().Add().SetAddress(ip.String()).SetPrefix(uint32(mask))

		// process nexthop
		if !ic.RetainNexthop {
			rrV4.SetNextHopMode(gosnappi.BgpV4RouteRangeNextHopMode.LOCAL_IP)
		} else {
			if err = imp.Processv4Nexthop(rrV4, nextHop, rre.Row); err != nil {
				rre.Err = &err
				return
			}
		}

		// process local Pref
		if err = imp.Processv4LocalPrf(rrV4, locPrf, rre.Row); err == nil {
			// process MED
			if err = imp.Processv4Metric(rrV4, metric, rre.Row); err == nil {
				// process origin
				var origin gosnappi.BgpRouteAdvancedOriginEnum
				if len(path) > 0 {
					if err, origin = getOriginValue(path[len(path)-1:]); err == nil {
						rrV4.Advanced().SetIncludeOrigin(true)
						rrV4.Advanced().SetOrigin(origin)
						// process ASPath
						err = imp.Processv4AsPath(rrV4, path, rre.Row)
					}
				} else {
					err = fmt.Errorf("found path parameter to be empty (line %d)", rre.Row+1)
				}
			}
		}
		if err != nil {
			rre.Err = &err
		} else {
			rre.RRv4 = rrV4
		}

		return
	}
}

func (imp *CiscoImporter) Processv4Nexthop(rr gosnappi.BgpV4RouteRange, nextHop string, row int) error {
	var ip net.IP
	if ip = net.ParseIP(nextHop); ip == nil {
		return fmt.Errorf("invalid ip address: %q for Nexthop processing (line %d)", nextHop, row+1)
	}
	if ip.To4() != nil {
		rr.SetNextHopMode(gosnappi.BgpV4RouteRangeNextHopMode.MANUAL)
		rr.SetNextHopIpv4Address(ip.String())
	} else {
		rr.SetNextHopMode(gosnappi.BgpV4RouteRangeNextHopMode.MANUAL)
		rr.SetNextHopIpv6Address(ip.String())
	}

	return nil
}

func (imp *CiscoImporter) Processv4LocalPrf(rr gosnappi.BgpV4RouteRange, token string, row int) error {
	if len(token) > 0 {
		if locprf, err := strconv.Atoi(token); err == nil {
			rr.Advanced().SetIncludeLocalPreference(true)
			rr.Advanced().SetLocalPreference(uint32(locprf))
		} else {
			return fmt.Errorf("invalid Local Pref: %q for processing (line %d) - %s", token, row+1, err.Error())
		}
	}

	return nil
}

func (imp *CiscoImporter) Processv4AsPath(rr gosnappi.BgpV4RouteRange, token string, row int) error {
	if len(token) <= 2 {
		// skip line, no as path
		return nil
	}

	token = token[:len(token)-2]
	if len(token) > 0 {
		token = strings.ReplaceAll(token, ",", " ")
		asPath := rr.AsPath()
		if imp.PeerV4.AsType() == gosnappi.BgpV4PeerAsType.EBGP {
			asPath.SetAsSetMode(gosnappi.BgpAsPathAsSetMode.INCLUDE_AS_SEQ)
		}
		asNums := strings.Fields(token)
		var last, cur gosnappi.BgpAsPathSegmentTypeEnum
		var err error = nil
		var index int = 0
		segNums := []uint32{}
		asSeg := asPath.Segments().Add()
		last = gosnappi.BgpAsPathSegmentType.AS_SEQ
		asSeg.SetType(gosnappi.BgpAsPathSegmentTypeEnum(last))
		for index < len(asNums) {
			numStr := asNums[index]
			newSegP, newSegN := false, false
			if cur, err = getAsPathSegType(numStr[0]); err != nil {
				return err
			}
			if last == gosnappi.BgpAsPathSegmentType.AS_SEQ {
				if cur != gosnappi.BgpAsPathSegmentType.AS_SEQ {
					newSegN = true
					numStr = numStr[1:]
					last = cur
				}
			} else if cur != gosnappi.BgpAsPathSegmentType.AS_SEQ {
				return fmt.Errorf("incorrect format of as path (line %d)", row+1)
			}
			if curT, err := getAsPathSegType(numStr[len(numStr)-1]); err != nil {
				return err
			} else if curT != gosnappi.BgpAsPathSegmentType.AS_SEQ {
				if last != curT {
					return fmt.Errorf("incorrect format of as path (line %d)", row+1)
				}
				newSegP = true
				numStr = numStr[:len(numStr)-1]
			}

			if newSegN {
				if len(segNums) > 0 {
					asSeg.SetAsNumbers(segNums)
					segNums = []uint32{}
					asSeg = asPath.Segments().Add()
				}
				asSeg.SetType(gosnappi.BgpAsPathSegmentTypeEnum(cur))
			}
			if asNum, err := strconv.Atoi(numStr); err != nil {
				return err
			} else {
				segNums = append(segNums, uint32(asNum))
			}
			if newSegP {
				asSeg.SetAsNumbers(segNums)
				segNums = []uint32{}
				if index+1 < len(asNums) {
					asSeg = asPath.Segments().Add()
					last = gosnappi.BgpAsPathSegmentType.AS_SEQ
				}
			}
			index++
		}
		if len(segNums) > 0 {
			asSeg.SetAsNumbers(segNums)
		}
	}

	return nil
}

func getAsPathSegType(b byte) (gosnappi.BgpAsPathSegmentTypeEnum, error) {
	switch b {
	case '{':
		fallthrough
	case '}':
		return gosnappi.BgpAsPathSegmentType.AS_SET, nil
	case '[':
		fallthrough
	case ']':
		return gosnappi.BgpAsPathSegmentType.AS_CONFED_SET, nil
	case '(':
		fallthrough
	case ')':
		return gosnappi.BgpAsPathSegmentType.AS_CONFED_SEQ, nil
	default:
		if b >= '0' && b <= '9' {
			return gosnappi.BgpAsPathSegmentType.AS_SEQ, nil
		}
	}
	return gosnappi.BgpAsPathSegmentType.AS_SEQ, fmt.Errorf("Invalid aspath segment marker %v", b)
}

func (imp *CiscoImporter) Processv4Metric(rr gosnappi.BgpV4RouteRange, token string, row int) error {
	if len(token) > 0 {
		if med, err := strconv.Atoi(token); err == nil {
			rr.Advanced().SetIncludeMultiExitDiscriminator(true)
			rr.Advanced().SetMultiExitDiscriminator(uint32(med))
		} else {
			return fmt.Errorf("invalid MED: %q for processing at row %d, error: %s", token, row+1, err.Error())
		}
	}

	return nil
}
func getOriginValue(origin string) (error, gosnappi.BgpRouteAdvancedOriginEnum) {
	origin = strings.Trim(origin, " ")
	if len(origin) > 0 {
		switch origin[0] {
		case 'i', 'I':
			return nil, gosnappi.BgpRouteAdvancedOrigin.IGP
		case 'e', 'E':
			return nil, gosnappi.BgpRouteAdvancedOrigin.EGP
		case '?':
			return nil, gosnappi.BgpRouteAdvancedOrigin.INCOMPLETE
		}
	}
	return fmt.Errorf("unknown origin string: %q", origin), gosnappi.BgpRouteAdvancedOrigin.INCOMPLETE
}

func ParseNetworkType(line string) (net.IP, error) {
	var ip net.IP
	splits := strings.Split(line, "/")
	cnt := len(splits)
	if cnt > 0 {
		if ip = net.ParseIP(splits[0]); ip == nil {
			return nil, fmt.Errorf("not valid ip address : %q", splits[0])
		}
		return ip, nil
	}

	return nil, fmt.Errorf("not valid network address : %q", line)
}

func ParseNetworkAddress(line string) (net.IP, int, error) {
	var ip net.IP
	var mask int
	var err error
	splits := strings.Split(line, "/")
	cnt := len(splits)
	if cnt > 0 {
		if ip = net.ParseIP(splits[0]); ip == nil {
			return nil, mask, fmt.Errorf("not valid ip address : %q", splits[0])
		}

		mask = 24
		if cnt > 1 {
			if mask, err = strconv.Atoi(splits[1]); err != nil {
				return nil, mask, err
			}
		}
	} else {
		return nil, mask, fmt.Errorf("not valid network address : %q", line)
	}

	return ip, mask, nil
}

func isSkippableLine(line *string) bool {
	// TBD:
	return false
}

func isValidRoute(line *string) bool {
	return (*line)[CISCO_VALID_ROUTE_OFFSET] == CISCO_VALID_ROUTE
}
