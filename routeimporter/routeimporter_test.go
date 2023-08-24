package routeimporter_test

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/open-traffic-generator/otg-route-importer/routeimporter"
	"github.com/open-traffic-generator/snappi/gosnappi"
)

func TestImportRoutesSmall(t *testing.T) {
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
		SessionName:   "txImp",
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
		SessionName:   "txImp",
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

	/*peer := ic.Targetv4Peers[0]
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
	}*/

	//t.Logf("\n***** importer: %v", is)
}

func TestImportRoutes1M(t *testing.T) {
	fmt.Printf("Number of cores: %d\n", runtime.NumCPU())

	is, err := routeimporter.GetImporterService(routeimporter.ImportFileTypeCisco)
	if err != nil {
		t.Errorf(fmt.Sprintf("Could not create Route Importer Service. Error: %v", err))
	}

	filename := "resource/cisco_v4_1M.txt"
	fb, err := os.ReadFile(filename)

	if err != nil {
		t.Errorf(fmt.Sprintf("Could not Read import file: %s. Error: %v", filename, err))
		return
	}

	ic := routeimporter.ImportConfig{
		SessionName:   "txImp",
		RRType:        routeimporter.RouteTypeIpv4,
		RetainNexthop: true,
		BestRoutes:    false,
		Targetv4Peers: []gosnappi.BgpV4Peer{gosnappi.NewBgpV4Peer()},
		Targetv6Peers: []gosnappi.BgpV6Peer{},
	}

	expRouteCount := 1000001
	names, err := is.ImportRoutes(ic, &fb)
	if err != nil {
		t.Errorf(fmt.Sprintf("Could not import routes. error: %v", err))
	} else if len(*names) != expRouteCount {
		t.Errorf("Could not successfully imported all routes. Expected Route Count: %d, Imported Routes Count: %d", expRouteCount, len(*names))
		fmt.Printf("imported routes name: %v", names)
	}
}
