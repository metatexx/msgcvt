package main

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/metatexx/avrox"
	must "github.com/metatexx/mxx/mustfatal"
)

func doAvroX(r io.Reader, basicShema string, unQuote, stripLF, quote bool, compressionType string) int {
	v := string(must.OkOne(io.ReadAll(r)))

	// Unquote the data if asked for
	if unQuote {
		if len(v) > 1 && v[0:1] != `"` {
			v = `"` + v + `"`
		}
		var errUnquote error
		v, errUnquote = strconv.Unquote(v)
		if errUnquote != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error while unquoting: %s\n", errUnquote)
			return 5
		}
	}

	// we strip a lf if asked for
	if stripLF && len(v) > 1 && v[len(v)-1:] == "\n" {
		v = v[:len(v)-1]
	}

	//fmt.Fprintf(os.Stderr, "%q", v)
	var data any
	switch basicShema {
	case "bytes":
		data = []byte(v)
	case "string":
		data = v
	case "int":
		var errAtoi error
		data, errAtoi = strconv.Atoi(v)
		if errAtoi != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error while unquoting: %s\n", errAtoi)
			return 5
		}
	}
	cID := avrox.CompNone
	switch compressionType {
	case "snappy", "snappy-block":
		cID = avrox.CompSnappy
	case "flate":
		cID = avrox.CompFlate
	case "gzip":
		cID = avrox.CompGZip
	}

	if quote {
		fmt.Printf("%q", must.OkOne(avrox.MarshalBasic(data, cID)))
	} else {
		fmt.Printf("%s", must.OkOne(avrox.MarshalBasic(data, cID)))
	}
	return 0
}
