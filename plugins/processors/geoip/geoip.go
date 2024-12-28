package geoip

import (
	"fmt"
	"net"

	"github.com/IncSW/geoip2"
	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/processors"
)

const sampleConfig = `
  ## city_db_path is the location of the MaxMind GeoIP2 City database
  city_db_path = "/var/lib/GeoIP/GeoLite2-City.mmdb"
  ## asn_db_path is the location of the MaxMind GeoIP2 ASN database
  asn_db_path = "/var/lib/GeoIP/GeoLite2-ASN.mmdb"

  [[processors.geoip.lookup]
	# get the ip from the field "source_ip" and put the lookup results in the respective destination fields (if specified)
	field = "source_ip"
	dest_country = "source_country"
	dest_city = "source_city"
	dest_lat = "source_lat"
	dest_lon = "source_lon"
  `

type lookupEntry struct {
	Field       string `toml:"field"`
	DestCountry string `toml:"dest_country"`
	DestCity    string `toml:"dest_city"`
	DestLat     string `toml:"dest_lat"`
	DestLon     string `toml:"dest_lon"`
	DestASN     string `toml:"dest_asn"`
	DestASNOrg  string `toml:"dest_asnorg"`
}

type GeoIP struct {
	CityDBPath string          `toml:"city_db_path"`
	ASNDBPath  string          `toml:"asn_db_path"`
	Lookups    []lookupEntry   `toml:"lookup"`
	Log        telegraf.Logger `toml:"-"`
}

var asnReader *geoip2.ASNReader
var cityReader *geoip2.CityReader
var countryReader *geoip2.CountryReader

func (g *GeoIP) SampleConfig() string {
	return sampleConfig
}

func (g *GeoIP) Description() string {
	return "GeoIP looks up the country code, city name and latitude/longitude for IP addresses in the MaxMind GeoIP database"
}

func (g *GeoIP) Apply(metrics ...telegraf.Metric) []telegraf.Metric {
	for _, point := range metrics {
		for _, lookup := range g.Lookups {
			if lookup.Field != "" {
				if value, ok := point.GetField(lookup.Field); ok {
					//if g.DBType == "city" || g.DBType == "" {
					ip := net.ParseIP(value.(string))
					if g.CityDBPath != "" {
						record, err := cityReader.Lookup(ip)
						if err != nil {
							if err.Error() != "not found" {
								g.Log.Errorf("GeoIP lookup error: %v", err)
							}
							continue
						}
						if len(lookup.DestCountry) > 0 {
							point.AddField(lookup.DestCountry, record.Country.ISOCode)
						}
						if len(lookup.DestCity) > 0 {
							point.AddField(lookup.DestCity, record.City.Names["en"])
						}
						if len(lookup.DestLat) > 0 {
							point.AddField(lookup.DestLat, record.Location.Latitude)
						}
						if len(lookup.DestLon) > 0 {
							point.AddField(lookup.DestLon, record.Location.Longitude)
						}
					}
					if g.ASNDBPath != "" {
						record, err := asnReader.Lookup(ip)
						if err != nil {
							if err.Error() != "not found" {
								g.Log.Errorf("GeoIP lookup error: %v", err)
							}
							continue
						}
						if len(lookup.DestCountry) > 0 {
							point.AddField(lookup.DestASN, record.AutonomousSystemNumber)
						}
						if len(lookup.DestCity) > 0 {
							point.AddField(lookup.DestASNOrg, record.AutonomousSystemOrganization)
						}
					}
					// }
					// else if g.DBType == "country" {
					// 	record, err := countryReader.Lookup(net.ParseIP(value.(string)))
					// 	if err != nil {
					// 		if err.Error() != "not found" {
					// 			g.Log.Errorf("GeoIP lookup error: %v", err)
					// 		}
					// 		continue
					// 	}
					// 	if len(lookup.DestCountry) > 0 {
					// 		point.AddField(lookup.DestCountry, record.Country.ISOCode)
					// 	}
					// } else {
					// 	g.Log.Errorf("Invalid GeoIP database type specified: %s", g.DBType)
					// }
				}
			}
		}
	}
	return metrics
}

func (g *GeoIP) Init() error {
	if g.CityDBPath != "" {
		r, err := geoip2.NewCityReaderFromFile(g.CityDBPath)
		if err != nil {
			return fmt.Errorf("error opening GeoIP city database: %v", err)
		} else {
			cityReader = r
		}
	}
	if g.ASNDBPath != "" {
		r, err := geoip2.NewASNReaderFromFile(g.ASNDBPath)
		if err != nil {
			return fmt.Errorf("error opening GeoIP ASN database: %v", err)
		} else {
			asnReader = r
		}
	}
	//  else if g.DBType == "country" {
	// 	r, err := geoip2.NewCountryReaderFromFile(g.DBPath)
	// 	if err != nil {
	// 		return fmt.Errorf("Error opening GeoIP database: %v", err)
	// 	} else {
	// 		countryReader = r
	// 	}
	// }
	return nil
}

func init() {
	processors.Add("geoip", func() telegraf.Processor {
		return &GeoIP{
			CityDBPath: "/var/lib/GeoIP/GeoLite2-Country.mmdb",
			ASNDBPath:  "/var/lib/GeoIP/GeoLite2-ASN.mmdb",
		}
	})
}
