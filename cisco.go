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

type CiscoImporter struct {
	id int64

	//
	POS_CISCO_HEADER_NETWORK  int
	POS_CISCO_HEADER_NEXT_HOP int
	POS_CISCO_HEADER_METRIC   int
	POS_CISCO_HEADER_LOC_PRF  int
	POS_CISCO_HEADER_WEIGHT   int
	POS_CISCO_HEADER_PATH     int

	//
	prefixLines  []string
	nextHopLines []string
	routeLines   []string

	//
	validRoutes int
	//routeRecords []RouteRecord
}

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
)

type RRInfo struct {
	Row    int
	TypeV4 bool // if IPv4 type
	RRv4   gosnappi.BgpV4RouteRange
	RRv6   gosnappi.BgpV6RouteRange
}

// String returns the id of the client.
func (imp *CiscoImporter) String() string {
	//return strconv.FormatInt(imp.id, 10)
	return fmt.Sprintf("Cisco Route Importer, session id: %8d, validRoutes:%d",
		imp.id, imp.validRoutes)
}

func (imp *CiscoImporter) ImportRoutesBuffer(ic ImportConfig, buffer []byte) error {

	// ParseAndPush
	/*
		OnParseAndPush();
		CheckForSize();
		PushToMW();
		ClearData();
	*/

	// OnParseAndPush
	{
		//defer profile.LogFuncDuration(, "ImportRoutesBuffer", "", "lineprocess")
		start := time.Now()

		var lineIndex int
		headerFound := false
		validCheck := false
		isLimitReached := false

		lineIndexList := []int{}

		if len(buffer) > 0 {

			startTask := time.Now()

			lines := strings.Split(string(buffer), "\n")

			// reader := bytes.NewReader(buffer)
			// better way to read incrementally??
			//for _, line := range lines {
			for i := 0; i < len(lines); i++ {
				line := lines[i]
				if res, err := imp.TryParseHeader(line); err == nil {
					if res {
						headerFound = true
						break
					}
					// continue to next line
				} else {
					return fmt.Errorf("cannot import, error:%v", err.Error())
				}
			}

			if !headerFound {
				return fmt.Errorf("cannot import, header not found")
			}
			//headerlines := lineIndex
			log.Info().Int64("miliseconds", time.Since(startTask).Milliseconds()).Msg("Header parsing")
			startTask = time.Now()

			pos := imp.POS_CISCO_HEADER_NETWORK
			var prevPrefix string

			// as headers are found look for data
			// for li, line := range lines {
			for i := lineIndex; i < len(lines); i++ {
				line := lines[i]
				// if li < headerlines {
				// 	continue
				// }
				space := 0
				lineIndex++

				if CheckForTabs(line) {
					return fmt.Errorf("contains tab character")
				}

				lineIndexList = append(lineIndexList, lineIndex)

				/*if (string.IsNullOrEmpty(line))
				{
					continue
				}*/
				if len(line) == 0 {
					continue
				}

				/*if (IsSkippableLine(line))
				{
					continue
				}*/

				if !IsValidRoute(line) {
					// TBD: SetInvalidMessage(lineIndexList, "Unsupported Format", out isLimitReached)
					continue
				}

				var prefix string
				if line[pos] == SPACE_CHAR {
					prefix = prevPrefix
				} else {
					//space = line.IndexOf(SPACE_CHAR, pos)
					space = strings.Index(line[pos:], " ")
					if space == -1 {
						//prefix = line.Substring(pos)
						prefix = line[pos:]
					} else {
						//prefix = line.Substring(pos, space-pos)
						prefix = line[pos:(space + pos)]
					}
					//log.Info().Msgf("Prefix: %s, space=%d, line=%q", prefix, space, line)
				}
				prevPrefix = prefix

				if !ProceedIfBestRouteIsSet(ic.BestRoutes, line) {
					continue
				}

				space = strings.Index(line[pos:], " ")
				if space == -1 {
					// line = NextLine(textReader)
					i++
					// TBD: add overflow check
					line = lines[i]
					lineIndex++
					lineIndexList = append(lineIndexList, lineIndex)
				}

				nextHop := ""
				posNextHop := imp.POS_CISCO_HEADER_NEXT_HOP

				if len(line) == 0 || len(line) <= posNextHop {
					// TBD: SetInvalidMessage(lineIndexList, "Unsupported Format", out isLimitReached)
					continue
				}

				//space = strings.Index(line[posNextHop:], string(SPACE_CHAR))
				space = strings.Index(line[posNextHop:], " ")

				//if (IsNextHopFromFile())
				if true {
					if space == -1 {
						//nextHop = line.Substring(posNextHop)
						nextHop = line[posNextHop:]
					} else {
						//nextHop = line.Substring(posNextHop, space - posNextHop)
						nextHop = line[posNextHop : space+posNextHop]
					}
				}

				if space == -1 {
					// line = NextLine(textReader)
					i++
					// TBD: add overflow check
					line = lines[i]
					lineIndex++
					lineIndexList = append(lineIndexList, lineIndex)
				}
				if len(line) == 0 {
					// TBD: SetInvalidMessage(lineIndexList, "Unsupported Format", out isLimitReached)
					continue
				}

				validCheck, isLimitReached = IsValidLine(prefix, nextHop, line, lineIndexList)
				if validCheck == false {
					continue
				}

				if isLimitReached {
					// TBD
				}

				imp.prefixLines = append(imp.prefixLines, prefix)
				imp.nextHopLines = append(imp.nextHopLines, nextHop)
				imp.routeLines = append(imp.routeLines, line)

				imp.validRoutes++

				/*if (m_prefixLines.Count == FILE_SPLIT_SIZE)
				{
					ProcessBatch();
				}
				if (GetValidRouteCount() == ((ulong)GetRouteLimit()))
				{
					break;
				}*/
			}

			log.Info().Int64("miliseconds", time.Since(startTask).Milliseconds()).Msg("line parsing")
			startTask = time.Now()

			/*if (m_prefixLines.Count > 0)
			  {
			      if (m_prefixLines.Count < m_routeRecords.Length)
			          m_routeRecords[m_prefixLines.Count] = null;
			      ProcessBatch();
			  }
			*/

			peer := gosnappi.NewBgpV4Peer()

			/*for ind := 0; ind < imp.validRoutes; ind++ {
				imp.ProcessLines(peer, ind)
			}*/

			if imp.validRoutes == 0 {
				log.Info().Msgf("*** No valid routes found")
				return nil
			}

			// Add place holder route ranges
			rrilist := []RRInfo{}
			startInd := 0
			rrType := ic.RRType
			/*if rrType == RouteTypeAuto {
				if err, rri := imp.AddRR(peer, startInd, rrType); err == nil {
					rrilist = append(rrilist, rri)
					startInd++
					if rri.TypeV4 {
						rrType = RouteTypeIpv4
					} else {
						rrType = RouteTypeIpv6
					}
				}
			}*/
			for ind := startInd; ind < imp.validRoutes; ind++ {
				if err, rri := imp.AddRR(peer, ind, rrType); err == nil {
					rrilist = append(rrilist, rri)
				}
			}

			log.Info().Int64("miliseconds", time.Since(startTask).Milliseconds()).Msg("Add routes")
			startTask = time.Now()

			/*for ind := 0; ind < imp.validRoutes; ind++ {
				peer.V4Routes().Add()
			}
			log.Info().Int64("miliseconds", time.Since(startTask).Milliseconds()).Msg("Add 1M routes")
			startTask = time.Now()*/

			// populate other attributes of RR
			useSequential := false
			if useSequential {
				for _, rri := range rrilist {
					imp.ProcessRR(rri, &ic)
				}
			} else {
				var wg sync.WaitGroup
				for _, rri := range rrilist {
					wg.Add(1)
					go func(rrii RRInfo) {
						defer wg.Done()
						imp.ProcessRR(rrii, &ic)
					}(rri)
				}
				wg.Wait()
			}

			log.Info().Int64("miliseconds", time.Since(startTask).Milliseconds()).Msg("Process routes")
			//startTask = time.Now()

			done := time.Since(start)
			log.Info().Int64("nanoseconds", done.Nanoseconds()).Msg("")

			//log.Info().Msgf("Importer Data (detail): %#v", imp)
			//log.Info().Msgf("Peer (detail): %#v", peer)
			if len(peer.V4Routes().Items()) < 100 {
				for _, rr := range peer.V4Routes().Items() {
					addr := rr.Addresses()
					for _, aa := range addr.Items() {
						log.Info().Msgf("RR-%q (detail): %v, %v, %v, Nexthop: %v", rr.Name(), aa.Address(), aa.Count(), aa.Prefix(), rr.NextHopIpv4Address())
					}
					if rr.Advanced().HasLocalPreference() {
						log.Info().Msgf("RR-%q (detail): LocalPre: %v", rr.Name(), rr.Advanced().LocalPreference())
					}
					if rr.Advanced().HasIncludeMultiExitDiscriminator() {
						log.Info().Msgf("RR-%q (detail): MED: %v", rr.Name(), rr.Advanced().MultiExitDiscriminator())
					}
					if rr.Advanced().HasIncludeOrigin() {
						log.Info().Msgf("RR-%q (detail): Origin: %v", rr.Name(), rr.Advanced().Origin())
					}

				}
			}

			log.Info().Msgf("Peer: v4 routes count = %d, v6 routes count = %d", len(peer.V4Routes().Items()), len(peer.V6Routes().Items()))

			return nil
		}
	}

	return fmt.Errorf("cannot import empty buffer : %q", buffer)
}

// check further -
func (imp *CiscoImporter) TryParseHeader(line string) (bool, error) {

	if IsHeaderLine(line) {
		if CheckForTabs(line) {
			return false, fmt.Errorf("contains tab character")
		}
		if _, err := imp.GetHeaderPositions(line); err != nil {
			return false, fmt.Errorf("cannot import, error:%v", err.Error())
		}
		return true, nil
	}
	return false, nil
}

func IsHeaderLine(line string) bool {
	return strings.HasPrefix(line, CISCO_HEADER_CHECK_STRING)
}

func CheckForTabs(line string) bool {
	return strings.ContainsAny(line, "\t")
}

func (imp *CiscoImporter) GetHeaderPositions(line string) (bool, error) {
	// pos := 0
	var pos int

	pos = strings.Index(line[pos:], CISCO_HEADER_NETWORK)
	if pos == -1 {
		return false, fmt.Errorf("header format is not proper")
	}
	imp.POS_CISCO_HEADER_NETWORK = pos

	/*if ((pos = i_headerLine.IndexOf(CISCO_HEADER_NEXT_HOP, pos + CISCO_HEADER_NETWORK.Length, StringComparison.Ordinal)) == -1)
	throw new InvalidDataException("Cisco header format is not proper.");
	POS_CISCO_HEADER_NEXT_HOP = pos;
	*/
	offset := pos + len(CISCO_HEADER_NETWORK)
	pos = strings.Index(line[offset:], CISCO_HEADER_NEXT_HOP)
	if pos == -1 {
		return false, fmt.Errorf("Cisco header format is not proper.")
	}
	pos = pos + offset
	imp.POS_CISCO_HEADER_NEXT_HOP = pos

	/*if ((pos = i_headerLine.IndexOf(CISCO_HEADER_METRIC, pos + CISCO_HEADER_NEXT_HOP.Length, StringComparison.Ordinal)) == -1) // as this is right aligned, so set the last index of header
	throw new InvalidDataException("Cisco header format is not proper.");
	POS_CISCO_HEADER_METRIC = pos + CISCO_HEADER_METRIC.Length;
	*/
	offset = pos + len(CISCO_HEADER_NEXT_HOP)
	pos = strings.Index(line[offset:], CISCO_HEADER_METRIC)
	if pos == -1 {
		return false, fmt.Errorf("Cisco header format is not proper.")
	}
	pos = pos + offset
	imp.POS_CISCO_HEADER_METRIC = pos + len(CISCO_HEADER_METRIC)

	/*if ((pos = i_headerLine.IndexOf(CISCO_HEADER_LOC_PRF, pos + CISCO_HEADER_METRIC.Length, StringComparison.Ordinal)) == -1) // as this is right aligned, so set the last index of header
	throw new InvalidDataException("Cisco header format is not proper.");
	imp.POS_CISCO_HEADER_LOC_PRF = pos + CISCO_HEADER_LOC_PRF.Length;*/
	offset = pos + len(CISCO_HEADER_METRIC)
	pos = strings.Index(line[offset:], CISCO_HEADER_LOC_PRF)
	if pos == -1 {
		return false, fmt.Errorf("Cisco header format is not proper.")
	}
	pos = pos + offset
	imp.POS_CISCO_HEADER_LOC_PRF = pos + len(CISCO_HEADER_LOC_PRF)

	/*if ((pos = i_headerLine.IndexOf(CISCO_HEADER_WEIGHT, pos + CISCO_HEADER_LOC_PRF.Length, StringComparison.Ordinal)) == -1) // as this is right aligned, so set the last index of header
	throw new InvalidDataException("Cisco header format is not proper.");
	imp.POS_CISCO_HEADER_WEIGHT = pos + CISCO_HEADER_WEIGHT.Length;*/
	offset = pos + len(CISCO_HEADER_LOC_PRF)
	pos = strings.Index(line[offset:], CISCO_HEADER_WEIGHT)
	if pos == -1 {
		return false, fmt.Errorf("Cisco header format is not proper.")
	}
	pos = pos + offset
	imp.POS_CISCO_HEADER_WEIGHT = pos + len(CISCO_HEADER_WEIGHT)

	/*if ((pos = i_headerLine.IndexOf(CISCO_HEADER_PATH, pos, StringComparison.Ordinal)) == -1)
	throw new InvalidDataException("Cisco header format is not proper.");
	imp.POS_CISCO_HEADER_PATH = pos;*/
	offset = pos + len(CISCO_HEADER_WEIGHT)
	pos = strings.Index(line[offset:], CISCO_HEADER_PATH)
	if pos == -1 {
		return false, fmt.Errorf("Cisco header format is not proper.")
	}
	pos = pos + offset
	imp.POS_CISCO_HEADER_PATH = pos

	return true, nil
}

func IsValidRoute(line string) bool {
	// if line[0] == CISCO_VALID_ROUTE {
	// 	return true
	// }
	// return false
	return line[0] == CISCO_VALID_ROUTE
}

func ProceedIfBestRouteIsSet(bestRouteOnly bool, line string) bool {
	if !bestRouteOnly {
		return true
	}

	if line[1] == CISCO_BEST_ROUTE {
		return true
	}
	return false
}

func IsValidLine(i_prefix string, i_nextHop string, i_line string, i_lineNumbersList []int) (bool, bool) {

	return true, true
	/*var i_isLimitReached bool
	prefix := i_prefix;
	nextHop := i_nextHop;
	line := i_line;
	shift := 0;

	if (!IsValidPrefix(prefix))
	{
		SetInvalidMessage(i_lineNumbersList, "Network Address", out i_isLimitReached);
		return false;
	}
	if (IsNextHopFromFile())
	{
		if (!IsValidIpAddress(i_nextHop, 0))
		{
			SetInvalidMessage(i_lineNumbersList, "Next Hop", out i_isLimitReached);
			return false;
		}
	}
	if (!IsValidNumeric(line, POS_CISCO_HEADER_METRIC))
	{
		SetInvalidMessage(i_lineNumbersList, "Metric", out i_isLimitReached);
		return false;
	}

	int posLocPrf = POS_CISCO_HEADER_LOC_PRF;
	int space = line.IndexOf(SPACE_CHAR, posLocPrf);
	if (space == -1)
	{
		shift = 0;
	}
	else
	{
		shift = space - posLocPrf;
	}

	if (!IsValidNumeric(line, POS_CISCO_HEADER_LOC_PRF + shift))
	{
		SetInvalidMessage(i_lineNumbersList, "Local Preference", out i_isLimitReached);
		return false;
	}
	if (!IsValidNumeric(line, POS_CISCO_HEADER_WEIGHT + shift))
	{
		SetInvalidMessage(i_lineNumbersList, "Weight", out i_isLimitReached);
		return false;
	}
	if (!IsValidAspath(line, POS_CISCO_HEADER_PATH + shift, SPACE_CHAR))
	{
		SetInvalidMessage(i_lineNumbersList, "AS Path", out i_isLimitReached);
		return false;
	}
	if (!IsValidOrigin(line))
	{
		SetInvalidMessage(i_lineNumbersList, "Origin", out i_isLimitReached);
		return false;
	}

	i_isLimitReached = false;
	return true;*/
}

func (imp *CiscoImporter) ProcessLines(peer gosnappi.BgpV4Peer, row int) error {

	// var pos int
	network := imp.prefixLines[row]
	//nextHop := imp.nextHopLines[row]
	//nextLine := imp.routeLines[row]

	if ip, mask, err := ParseNetworkAddress(network); err == nil {
		//ipType := "IPv4"
		v4Type := true
		if ip.To4() == nil {
			//ipType = "IPv6"
			v4Type = false
		}

		if v4Type {
			var rr gosnappi.BgpV4RouteRange
			rr = peer.V4Routes().Add()
			rr.Addresses().Add().SetAddress(ip.String()).SetPrefix(int32(mask))

			// process other attributes
			// process nexthop
			// process nextline
		} else {
			var rr gosnappi.BgpV6RouteRange
			rr = peer.V6Routes().Add()
			rr.Addresses().Add().SetAddress(ip.String()).SetPrefix(int32(mask))

			// process other attributes
		}

		//log.Info().Msgf("Row: %d, Network Address:%s/%d(%s),", row, ip.String(), mask, ipType)
	} else {
		log.Info().Msgf("Row: %d, Network Address parsing error:%s", row, err.Error())
	}

	return nil
}

func (imp *CiscoImporter) AddRR(peer gosnappi.BgpV4Peer, row int, routeType RouteType) (error, RRInfo) {

	finalRRtype := routeType
	if routeType == RouteTypeAuto {
		network := imp.prefixLines[row]

		if ip, err := ParseNetworkType(network); err == nil {
			finalRRtype = RouteTypeIpv4
			if ip.To4() == nil {
				finalRRtype = RouteTypeIpv6
			}
		} else {
			log.Info().Msgf("Row: %d, Address: %q, Network Address parsing error:%s", row, network, err.Error())
			return err, RRInfo{}
		}
	}

	switch finalRRtype {
	case RouteTypeIpv4:
		rr := peer.V4Routes().Add()
		rrInfo := RRInfo{
			TypeV4: true,
			Row:    row,
			RRv4:   rr,
		}
		return nil, rrInfo
	case RouteTypeIpv6:
		rr := peer.V6Routes().Add()
		rrInfo := RRInfo{
			TypeV4: false,
			Row:    row,
			RRv6:   rr,
		}
		return nil, rrInfo
	}

	/*
		network := imp.prefixLines[row]
		//nextHop := imp.nextHopLines[row]
		//nextLine := imp.routeLines[row]

		if ip, err := ParseNetworkType(network); err == nil {
			//ipType := "IPv4"
			v4Type := true
			if ip.To4() == nil {
				//ipType = "IPv6"
				v4Type = false
			}

			if v4Type {
				var rr gosnappi.BgpV4RouteRange
				rr = peer.V4Routes().Add()

				rrInfo := RRInfo{
					TypeV4: true,
					Row:    row,
					RRv4:   rr,
				}
				// rr.Addresses().Add().SetAddress(ip.String()).SetPrefix(int32(mask))

				// process other attributes
				// process nexthop
				// process nextline
				return nil, rrInfo
			} else {
				var rr gosnappi.BgpV6RouteRange
				rr = peer.V6Routes().Add()
				rrInfo := RRInfo{
					TypeV4: false,
					Row:    row,
					RRv6:   rr,
				}
				// rr.Addresses().Add().SetAddress(ip.String()).SetPrefix(int32(mask))

				// process other attributes
				return nil, rrInfo
			}

			//log.Info().Msgf("Row: %d, Network Address:%s/%d(%s),", row, ip.String(), mask, ipType)
		} else {
			log.Info().Msgf("Row: %d, Network Address parsing error:%s", row, err.Error())
			return err, RRInfo{}
		}*/

	return nil, RRInfo{}
}

func (imp *CiscoImporter) ProcessRR(rri RRInfo, ic *ImportConfig) error {

	row := rri.Row
	// var pos int
	network := imp.prefixLines[row]
	nextLine := imp.routeLines[row]

	/*nextHop := imp.nextHopLines[row]
	nextLine := imp.routeLines[row]

	log.Info().Msgf("*** Row: %d, NextHop: %q, NextLine: %q", row, nextHop, nextLine)*/

	if ip, mask, err := ParseNetworkAddress(network); err == nil {
		//ipType := "IPv4"
		v4Type := true
		if ip.To4() == nil {
			//ipType = "IPv6"
			v4Type = false
		}

		if v4Type {
			//var rr gosnappi.BgpV4RouteRange
			//rr = peer.V4Routes().Add()
			rr := rri.RRv4
			rr.SetName(fmt.Sprintf("Imported-%d", row))
			rr.Addresses().Add().SetAddress(ip.String()).SetPrefix(int32(mask))

			// process nexthop
			if ic.LocalNexthop {
				rr.SetNextHopMode(gosnappi.BgpV4RouteRangeNextHopMode.LOCAL_IP)
			} else {
				imp.Processv4Nexthop(rr, row)
			}

			// process local Pref
			imp.Processv4LocalPrf(rr, row)

			// process MED
			imp.Processv4Metric(rr, row)

			// process ASPath
			//imp.Processv4AsPath(rr, row)

			// process origin
			if err, origin := getOriginValue(nextLine[len(nextLine)-1:]); err == nil {
				rr.Advanced().SetIncludeOrigin(true)
				rr.Advanced().SetOrigin(origin)
			} else {
				// print error
			}
			// process nextline

		} else {
			//var rr gosnappi.BgpV6RouteRange
			//rr = peer.V6Routes().Add()
			rr := rri.RRv6
			rr.SetName(fmt.Sprintf("Imported-%d", row))
			rr.Addresses().Add().SetAddress(ip.String()).SetPrefix(int32(mask))

			// process nexthop
			if ic.LocalNexthop {
				rr.SetNextHopMode(gosnappi.BgpV6RouteRangeNextHopMode.LOCAL_IP)
			} else {
				imp.Processv6Nexthop(rr, row)
			}
		}

		//log.Info().Msgf("Row: %d, Network Address:%s/%d(%s),", row, ip.String(), mask, ipType)
	} else {
		log.Info().Msgf("Row: %d, Network Address parsing error:%s", row, err.Error())
	}

	return nil
}

func (imp *CiscoImporter) Processv4Nexthop(rr gosnappi.BgpV4RouteRange, row int) error {
	nextHop := imp.nextHopLines[row]

	var ip net.IP
	if ip = net.ParseIP(nextHop); ip == nil {
		return fmt.Errorf("invalid ip address: %q for Nexthop processing at row %d", nextHop, row)

	} else {
		v4Type := true
		if ip.To4() == nil {
			v4Type = false
		}

		if v4Type {
			rr.SetNextHopMode(gosnappi.BgpV4RouteRangeNextHopMode.MANUAL)
			rr.SetNextHopIpv4Address(ip.String())
		} else {
			rr.SetNextHopMode(gosnappi.BgpV4RouteRangeNextHopMode.MANUAL)
			rr.SetNextHopIpv6Address(ip.String())
		}
	}

	return nil
}

func (imp *CiscoImporter) Processv6Nexthop(rr gosnappi.BgpV6RouteRange, row int) error {
	nextHop := imp.nextHopLines[row]

	var ip net.IP
	if ip = net.ParseIP(nextHop); ip == nil {
		return fmt.Errorf("invalid ip address: %q for Nexthop processing at row %d", nextHop, row)

	} else {
		v4Type := true
		if ip.To4() == nil {
			v4Type = false
		}

		if v4Type {
			rr.SetNextHopMode(gosnappi.BgpV6RouteRangeNextHopMode.MANUAL)
			rr.SetNextHopIpv4Address(ip.String())
		} else {
			rr.SetNextHopMode(gosnappi.BgpV6RouteRangeNextHopMode.MANUAL)
			rr.SetNextHopIpv6Address(ip.String())
		}
	}

	return nil
}

func (imp *CiscoImporter) Processv4LocalPrf(rr gosnappi.BgpV4RouteRange, row int) error {
	//nextHop := imp.nextHopLines[row]
	nextLine := imp.routeLines[row]

	token := nextLine[imp.POS_CISCO_HEADER_METRIC:imp.POS_CISCO_HEADER_LOC_PRF]
	token = strings.Trim(token, " ")
	if len(token) > 0 {
		if locprf, err := strconv.Atoi(token); err == nil {
			//return nil, mask, err
			rr.Advanced().SetIncludeLocalPreference(true)
			rr.Advanced().SetLocalPreference(int32(locprf))
		} else {
			return fmt.Errorf("invalid Local Pref: %q for processing at row %d, error: %s", token, row, err.Error())
		}
	}

	return nil
}

func (imp *CiscoImporter) Processv4AsPath(rr gosnappi.BgpV4RouteRange, row int) error {
	nextLine := imp.routeLines[row]

	token := nextLine[imp.POS_CISCO_HEADER_PATH : len(nextLine)-2]
	token = strings.Trim(token, " ")
	log.Info().Msgf("Row: %d, aspath:%s", row, token)
	/*if len(token) > 0 {
		if locprf, err := strconv.Atoi(token); err == nil {
			//return nil, mask, err
			rr.Advanced().SetIncludeLocalPreference(true)
			rr.Advanced().SetLocalPreference(int32(locprf))
		} else {
			return fmt.Errorf("invalid Local Pref: %q for processing at row %d, error: %s", token, row, err.Error())
		}
	}*/

	return nil
}

func (imp *CiscoImporter) Processv4Metric(rr gosnappi.BgpV4RouteRange, row int) error {
	nextLine := imp.routeLines[row]

	token := nextLine[imp.POS_CISCO_HEADER_NEXT_HOP:imp.POS_CISCO_HEADER_METRIC]
	//tokens := strings.Split(token, " ")
	tokens := strings.Fields(token)
	medstr := ""
	if len(tokens) > 1 {
		medstr = tokens[1]
		//medstr = strings.Trim(medstr, " ")

	}
	//log.Info().Msgf("Row: %d, token:%q, med:%s", row, token, medstr)
	if len(medstr) > 0 {
		if med, err := strconv.Atoi(medstr); err == nil {
			//return nil, mask, err
			rr.Advanced().SetIncludeMultiExitDiscriminator(true)
			rr.Advanced().SetMultiExitDiscriminator(int32(med))
		} else {
			return fmt.Errorf("invalid MED: %q for processing at row %d, error: %s", token, row, err.Error())
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
	//var err error
	//var addr string
	splits := strings.Split(line, "/")
	cnt := len(splits)
	if cnt > 0 {
		//addr = splits[0]
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
		//log.Info().Msgf("IP: %#v", ip)
		mask = 24
		if cnt > 1 {
			if mask, err = strconv.Atoi(splits[1]); err != nil {
				return nil, mask, err
			}
		}

		//log.Info().Msgf("Mask: %#v", mask)
	} else {
		return nil, mask, fmt.Errorf("not valid network address : %q", line)
	}

	return ip, mask, nil
}