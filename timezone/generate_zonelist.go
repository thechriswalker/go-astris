// +build ignore

// Theres a bit of a problem with most iana timezone lookup code, that
// while allowing a lookup of a specific code, there is no way to enumerate
// them.
// This code pulls and generates the list of location based time zone names
// from the web and parses it into go code
package main

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"time"
)

const timezoneZipURL = "https://timezonedb.com/files/timezonedb.csv.zip"

func main() {
	resp, err := http.Get(timezoneZipURL)
	if err != nil {
		panic(err)
	}

	// buffer the whole thing!
	var buf bytes.Buffer
	defer resp.Body.Close()
	_, err = io.Copy(&buf, resp.Body)
	if err != nil {
		panic(err)
	}
	b := buf.Bytes()
	l := int64(len(b))
	// unzip as needed
	zr, err := zip.NewReader(bytes.NewReader(b), l)
	if err != nil {
		panic(err)
	}

	// read the "zone.csv" file through a CSV parser
	zoneCsv, err := zr.Open("zone.csv")
	if err != nil {
		panic(err)
	}
	rd := csv.NewReader(zoneCsv)

	zones := make([]string, 0, 800)
	for {
		row, err := rd.Read()
		// the 3rd field contains the data we want.
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		zones = append(zones, row[2])
	}

	// sort to make the prefix search easy
	sort.StringSlice(zones).Sort()

	// and finalise the file
	// lets write the header of our file first.
	out, err := os.Create("zonelist.go")
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(out, `package timezone

// List of Location based timezone names
// from: %s
//   at: %s
var List = []string{
`, timezoneZipURL, time.Now().Format(time.RFC3339))

	for _, s := range zones {
		fmt.Fprintf(out, "	%q,\n", s)
	}

	out.WriteString("}\n")
}
