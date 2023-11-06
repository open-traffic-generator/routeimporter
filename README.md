# OTG Route Importer
OTG route importer is an utility library for importing BGP routes that are saved in vendor specific format to Open Traffic Generator configuration. Current support is for output of Cisco IOS specific show routes output(extendable to any vendor specific route format). Users can add support and extend the library for other vendor specific file formats.


## Start Using
The code snippet, in golang, illustrates how the package can be used to import BGP routes that is saved by Cisco router, and updated in the configuration. 

```
	 //----------------------------------------------------------------------------
	 // Import route from file
	 //----------------------------------------------------------------------------
 	 device := config.Devices().Items()[0]
	 txPeerBgp := device.Bgp()
	 ipv4InterfaceBgp4 := txPeerBgp.Ipv4Interfaces().Items()[0]
	 txPeer := ipv4InterfaceBgp4.Peers().Items()[0]
	 filename := "./cisco_v4_1M.txt"

	 fb, err := os.ReadFile(filename)
	 if err != nil {
		 // Error
	 }

	 // ImportConfig defines various import parameters that dictates the import processing
	 ic := routeimporter.ImportConfig{
		 NamePrefix:       "txImp",
		 RRType:           routeimporter.RouteTypeIpv4,
		 BestRoutes:       true,
		 RetainNexthop:    false,
		 SequentialProcess false,
		 Targetv4Peers:    []gosnappi.BgpV4Peer{txPeer},
		 Targetv6Peers:    []gosnappi.BgpV6Peer{},
	 }

	 // Get route importer service specific to the import file format
	 is, err := routeimporter.GetImporterService(routeimporter.ImportFileTypeCisco)
	 if err != nil {
		 // Error
	 }

	 // Import routes based on the config. On success Targetv4Peers / Targetv6Peers
         // gets updated with valid routes.
	 _, err := is.ImportRoutes(ic, &fb)
	 if err != nil {
		 // Error
	 }
	 //----------------------------------------------------------------------------
	 // End import route from file
	 //----------------------------------------------------------------------------

	 routes := txPeer.V4Routes().Items()
	 fmt.Printf("Number of routes imported = %v\n", len(routes))
```

## For development
   The package can be extended to support other vendor formats. Add a new import service in routeimporter.go for each new vendor. ImportRoutes api for new import service is expected to process route import file and update the target BGP peer with valid routes. Developers can add new additional config parameters in api.go definition.  
