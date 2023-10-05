package routeimporter_test

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/open-traffic-generator/routeimporter"
	"github.com/open-traffic-generator/snappi/gosnappi"
)

type REntry struct {
	Address string
	Prefix  uint32
	NextHop string
	Path    string
}

func TestImportRoutesPopulateConfig(t *testing.T) {
	api := gosnappi.NewApi()

	config := api.NewConfig()

	dta := config.Devices().Add().SetName("devA")

	txPeer := dta.Bgp().SetRouterId("1.1.1.1").Ipv4Interfaces().Add().SetIpv4Name("intA").Peers().Add().SetName("peerA")

	txPeer.SetPeerAddress("1.1.1.1").SetAsType(gosnappi.BgpV4PeerAsType.EBGP)
	txgroup := txPeer.V4RouteGroups().Add().SetName("groupTxV4")

	if js, err := config.ToJson(); err == nil {
		fmt.Printf("Config Before Import: %v", js)
	} else {
		t.Errorf("failed to convert config in Json format. Error: %v", err.Error())
	}

	filename := "resource/cisco_v4_basic.txt"
	fb, err := os.ReadFile(filename)
	if err != nil {
		t.Errorf(fmt.Sprintf("Could not Read import file: %s. Error: %v", filename, err))
		return
	}

	ic := routeimporter.ImportConfig{
		NamePrefix:    "txImp",
		RRType:        routeimporter.RouteTypeIpv4,
		RetainNexthop: true,
		BestRoutes:    true,
		Targetv4Peers: []gosnappi.BgpV4Peer{txPeer},
		Targetv6Peers: []gosnappi.BgpV6Peer{},
	}

	is, err := routeimporter.GetImporterService(routeimporter.ImportFileTypeCisco)
	if err != nil {
		t.Errorf(fmt.Sprintf("Could not create Route Importer Service. Error: %v", err))
	}
	names, err := is.ImportRoutes(ic, &fb)
	if err != nil {
		t.Errorf(fmt.Sprintf("Could not import routes. error: %v", err))
	}
	txgroup.SetRouteNames(*names)

	// validate route count
	expRouteCount := 3
	if len(txPeer.V4Routes().Items()) != expRouteCount {
		t.Errorf("Could not successfully imported all routes. Expected Route Count: %d, Imported Routes Count: %d",
			expRouteCount, len(txPeer.V4Routes().Items()))
	}
	if js, err := config.ToJson(); err == nil {
		fmt.Printf("Config After Import: %v", js)
	} else {
		t.Errorf("failed to convert config in Json format. Error: %v", err.Error())
	}
}

func TestImportRoutesSimpleV4(t *testing.T) {
	filename := "resource/cisco_v4_basic.txt"

	is, err := routeimporter.GetImporterService(routeimporter.ImportFileTypeCisco)
	if err != nil {
		t.Errorf(fmt.Sprintf("Could not create Route Importer Service. Error: %v", err))
	}

	fb1, err := os.ReadFile(filename)
	if err != nil {
		t.Errorf(fmt.Sprintf("Could not Read import file: %s. Error: %v", filename, err))
		return
	}

	ic := routeimporter.ImportConfig{
		NamePrefix:    "txImp",
		RRType:        routeimporter.RouteTypeIpv4,
		RetainNexthop: true,
		BestRoutes:    true,
		Targetv4Peers: []gosnappi.BgpV4Peer{gosnappi.NewBgpV4Peer()},
		Targetv6Peers: []gosnappi.BgpV6Peer{},
	}

	expRouteCount := 3
	names, err := is.ImportRoutes(ic, &fb1)
	if err != nil {
		t.Errorf(fmt.Sprintf("Could not import routes. error: %v", err))
	} else if len(*names) != expRouteCount {
		t.Errorf("Could not successfully imported all routes. Expected Route Count: %d, Imported Routes Count: %d", expRouteCount, len(*names))
		fmt.Printf("**** routes imported: %v", names)
	}

	is, err = routeimporter.GetImporterService(routeimporter.ImportFileTypeCisco)
	if err != nil {
		t.Errorf(fmt.Sprintf("Could not create Route Importer Service. Error: %v", err))
	}
	fb2, err := os.ReadFile(filename)
	if err != nil {
		t.Errorf(fmt.Sprintf("Could not Read import file: %s. Error: %v", filename, err))
		return
	}
	ic = routeimporter.ImportConfig{
		NamePrefix:    "txImp",
		RRType:        routeimporter.RouteTypeIpv4,
		RetainNexthop: true,
		BestRoutes:    false,
		Targetv4Peers: []gosnappi.BgpV4Peer{gosnappi.NewBgpV4Peer()},
		Targetv6Peers: []gosnappi.BgpV6Peer{},
	}

	expRouteCount = 6
	names, err = is.ImportRoutes(ic, &fb2)
	if err != nil {
		t.Errorf(fmt.Sprintf("Could not import routes. error: %v", err))
	} else if len(*names) != expRouteCount {
		t.Errorf("Could not successfully imported all routes. Expected Route Count: %d, Imported Routes Count: %d", expRouteCount, len(*names))
		fmt.Printf("imported routes name: %v", names)
	}
}

func TestImportRoutes1KV4(t *testing.T) {
	fmt.Printf("Number of cores: %d\n", runtime.NumCPU())

	is, err := routeimporter.GetImporterService(routeimporter.ImportFileTypeCisco)
	if err != nil {
		t.Errorf(fmt.Sprintf("Could not create Route Importer Service. Error: %v", err))
	}

	filename := "resource/cisco_v4_1K.txt"
	fb, err := os.ReadFile(filename)

	if err != nil {
		t.Errorf(fmt.Sprintf("Could not Read import file: %s. Error: %v", filename, err))
		return
	}

	ic := routeimporter.ImportConfig{
		NamePrefix:    "txImp",
		RRType:        routeimporter.RouteTypeIpv4,
		RetainNexthop: true,
		BestRoutes:    false,
		Targetv4Peers: []gosnappi.BgpV4Peer{gosnappi.NewBgpV4Peer()},
		Targetv6Peers: []gosnappi.BgpV6Peer{},
	}

	expRouteCount := 1000
	names, err := is.ImportRoutes(ic, &fb)
	if err != nil {
		t.Errorf(fmt.Sprintf("Could not import routes. error: %v", err))
	} else if len(*names) != expRouteCount {
		t.Errorf("Could not successfully imported all routes. Expected Route Count: %d, Imported Routes Count: %d", expRouteCount, len(*names))
		fmt.Printf("imported routes name: %v", names)
	}

	rEntryList := []REntry{
		{Address: "1.0.141.0", Prefix: 24, NextHop: "203.119.104.2", Path: "[4608,6939,38040,23969,23969]"},
		{Address: "1.0.144.0", Prefix: 20, NextHop: "203.119.104.1", Path: "[4608,4651,23969]"},
		{Address: "1.0.160.0", Prefix: 19, NextHop: "202.12.28.1", Path: "[4777,6939,38040,23969]"},
		{Address: "1.0.169.0", Prefix: 24, NextHop: "203.119.104.1", Path: "[4608,6939,38040,23969]"},
		{Address: "1.6.134.0", Prefix: 23, NextHop: "203.119.104.1", Path: "[4608,24115,9583]"},
	}

	for _, rr := range ic.Targetv4Peers[0].V4Routes().Items() {
		addr := rr.Addresses().Items()[0]
		path := rr.AsPath().Segments().Items()[0]
		pathStr := strings.Join(strings.Fields(fmt.Sprint(path.AsNumbers())), ",")
		for i, entry := range rEntryList {
			if addr.Address() == entry.Address &&
				addr.Prefix() == entry.Prefix &&
				rr.NextHopIpv4Address() == entry.NextHop &&
				pathStr == entry.Path {
				rEntryList = append(rEntryList[:i], rEntryList[i+1:]...)
			}
		}
	}
	if len(rEntryList) > 0 {
		t.Errorf("Could not successfully imported all routes. Number of missing routes found: %d", len(rEntryList))
	}
}
