package routeimporter_test

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/open-traffic-generator/otg-route-importer/routeimporter"
)

func TestCreateService(t *testing.T) {

	is, err := routeimporter.GetImporterService(routeimporter.ImportFileTypeCisco)
	if err != nil {
		t.Errorf(fmt.Sprintf("Could not initialize Route Importer: %v", err))
	}

	fb, err := os.ReadFile("resource/cisco_v4_basic.txt")

	if err != nil {
		t.Error(err)
		return
	}

	ic := routeimporter.ImportConfig{RRType: routeimporter.RouteTypeIpv4, RetainNexthop: true, BestRoutes: true}
	err = is.ImportRoutesBuffer(ic, fb)
	if err != nil {
		t.Errorf(fmt.Sprintf("Could not import buffer: %v, error: %v", fb, err))
	}

	//fmt.Printf("importer: %v", is)
}

func TestCreate1MService(t *testing.T) {
	fmt.Printf("Number of cores: %d\n", runtime.NumCPU())

	is, err := routeimporter.GetImporterService(routeimporter.ImportFileTypeCisco)
	if err != nil {
		t.Errorf(fmt.Sprintf("Could not initialize Route Importer: %v", err))
	}

	fb, err := os.ReadFile("resource/cisco_v4_1M.txt")

	if err != nil {
		t.Error(err)
		return
	}

	//ic := routeimport.ImportConfig{RRType: routeimport.RouteTypeAuto}
	ic := routeimporter.ImportConfig{RRType: routeimporter.RouteTypeIpv4, RetainNexthop: true, BestRoutes: false}
	err = is.ImportRoutesBuffer(ic, fb)
	if err != nil {
		t.Errorf(fmt.Sprintf("Could not import buffer: %v, error: %v", fb, err))
	}

	//fmt.Printf("importer: %v", is)
}
