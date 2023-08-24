package routeimporter_test

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/open-traffic-generator/otg-route-importer/routeimporter"
	"github.com/open-traffic-generator/snappi/gosnappi"
)

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
		SessionName:   "txImp",
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

func TestImportRoutes1MV4(t *testing.T) {
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
