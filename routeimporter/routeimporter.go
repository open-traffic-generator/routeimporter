package routeimporter

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

var gid int64 = 0

func newCiscoImporter() (ImportService, error) {
	gid += 1
	is := &CiscoImporter{
		id: gid,
	}
	log.Info().Msgf("CiscoImporter: %v created", is)

	return is, nil
}

func GetImporterService(format ImportFileType) (ImportService, error) {
	switch format {
	case ImportFileTypeCisco:
		return newCiscoImporter()
	case ImportFileTypeJuniper:
		return nil, fmt.Errorf("support for Juniper not yet implemented")
	}

	return nil, fmt.Errorf("not supported / unknown format : %v", format)
}
