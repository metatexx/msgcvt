package main

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/fxamacker/cbor/v2"
	"github.com/golang/snappy"
	"github.com/hamba/avro/v2"
	must "github.com/metatexx/mxx/mustfatal"
)

func main() {
	flagSnappyBlock := flag.Bool("snappy-block", false, "decode data with snappy (block mode) first")
	flagSnappyStream := flag.Bool("snappy-stream", false, "decode data with snappy (stream mode) first")
	flagGZip := flag.Bool("gzip", false, "decode data with GZip first")
	flagDeflate := flag.Bool("deflate", false, "decode data with deflate first")
	flagString := flag.Bool("string", false, "output raw as escaped string")
	flagCBOR := flag.Bool("cbor", false, "output CBOR as JSON")
	flagGOB := flag.Bool("gob", false, "output GOB as JSON")
	flagAVRO := flag.String("avro", "", "avro base schema (will also set From to 'avro' format and To as json 'format')")
	flag.Parse()

	var r io.Reader
	r = os.Stdin

	if *flagSnappyBlock {
		// There is no streaming decompression for snappy
		// read all data
		compressed := must.OkOne(io.ReadAll(r))
		// decompress it
		decompressed, errDecompress := snappy.Decode(nil, compressed)
		if errDecompress == nil {
			// make a new reader from that buffer
			r = bytes.NewBuffer(decompressed)
		} else {
			// if we can't decompress, we just leave it as it is
			// this may be "right"
			r = bytes.NewBuffer(compressed)
		}
	}

	if *flagSnappyStream {
		r = snappy.NewReader(r)
	}

	if *flagGZip {
		r = must.OkOne(gzip.NewReader(r))
	}
	if *flagDeflate {
		r = flate.NewReader(r)
	}

	if *flagString {
		var buf bytes.Buffer
		must.OkSkipOne(buf.ReadFrom(r))
		fmt.Printf("%#v\n", buf.String())
	} else if *flagCBOR {
		dec := cbor.NewDecoder(r)
		var cborNative any
		must.Ok(dec.Decode(&cborNative))
		mappedKeys := must.OkOne(convertMapKeysToStrings(cborNative))
		jsonData := must.OkOne(json.MarshalIndent(mappedKeys, "", "  "))
		fmt.Printf("%s\n", jsonData)
	} else if *flagGOB {
		dec := gob.NewDecoder(r)
		// Decode the gob-encoded map
		var gobNative map[string]interface{}
		must.Ok(dec.Decode(&gobNative))
		// Convert the decoded data to JSON
		jsonData := must.OkOne(json.MarshalIndent(gobNative, "", "  "))
		fmt.Printf("%s\n", jsonData)
	} else if *flagAVRO != "" {
		schemaString := string(must.OkOne(os.ReadFile(*flagAVRO)))
		schema := must.OkOne(avro.Parse(schemaString))
		dec := avro.NewDecoderForSchema(schema, r)
		var avroNative any
		must.Ok(dec.Decode(&avroNative))
		jsonData := must.OkOne(json.MarshalIndent(avroNative, "", "  "))
		fmt.Printf("%s\n", jsonData)
	} else {
		must.OkSkipOne(io.Copy(os.Stdout, r))
		//fmt.Println("unsupported format conversion")
	}
}

func convertMapKeysToStrings(data interface{}) (map[string]interface{}, error) {
	if m, ok1 := data.(map[interface{}]interface{}); ok1 {
		strMap := make(map[string]interface{})
		for k, v := range m {
			if strKey, ok2 := k.(string); ok2 {
				strMap[strKey] = v
			} else {
				return nil, fmt.Errorf("cannot convert map key %v to string", k)
			}
		}
		return strMap, nil
	}
	return nil, fmt.Errorf("input data is not a map")
}
