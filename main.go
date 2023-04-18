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
	"strconv"
	"strings"

	"github.com/fxamacker/cbor/v2"
	"github.com/golang/snappy"
	"github.com/hamba/avro/v2"
	"github.com/metatexx/avrox"
	must "github.com/metatexx/mxx/mustfatal"
)

func main() {
	defer func() {
		rv := recover()
		if rv != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Recovered from panic: %v\n", rv)
			os.Exit(5)
		}
	}()

	flagAnalyse := flag.Bool("analyse", false, "pipes data and gives some info about it")
	flagFile := flag.String("f", "", "read from the given file")
	flagSnappyBlock := flag.Bool("snappy", false, "encode/decode data with snappy (block mode)")
	flagSnappyStream := flag.Bool("snappy-stream", false, "encode/decode data with snappy (stream mode)")
	flagGZip := flag.Bool("gzip", false, "encode/decode data with GZip (default compression)")
	flagFlate := flag.Bool("flate", false, "encode/decode data with deflate (default compression)")
	flagQuote := flag.Bool("q", false, "quote output string (escapes)")
	flagCBOR := flag.Bool("cbor", false, "output CBOR as JSON")
	flagGOB := flag.Bool("gob", false, "output GOB as JSON")
	flagAvro := flag.String("avro", "", "avro base schema (will also set From to 'avro' format and To as json 'format')")
	flagSkipAvroX := flag.Bool("skip-avrox", false, "don't check for avrox in raw mode")
	flagAvroX := flag.String("avrox", "", "create a AvroX basic string|int|[]byte from input")
	flagHandleLF := flag.Bool("n", false,
		"don't add a lf at the end of avrox outputs and strip it from inputs (avrox specific)")
	flagUnquote := flag.Bool("e", false, "unquotes arguments")
	flag.Parse()

	var r io.Reader
	if *flagFile != "" {
		r = must.OkOne(os.Open(*flagFile))
	} else {
		r = os.Stdin
	}

	if flag.NArg() == 1 {
		r = strings.NewReader(flag.Arg(0))
	}

	if *flagAnalyse {
		b := make([]byte, 4)
		n := must.IgnoreOne(r.Read(b))
		if n == 0 {
			_, _ = fmt.Fprintf(os.Stderr, "0 bytes\n")
			os.Exit(0)
		}
		if avrox.IsMagic(b) {
			nID, sID, cID, err := avrox.DecodeMagic(b)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "err: %s", err)
				os.Exit(5)
			}
			var size1 int
			var size2 int64
			size1, err = os.Stdout.Write(b)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "err: %s", err)
				os.Exit(5)

			}
			size2, err = io.Copy(os.Stdout, r)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "err: %s", err)
				os.Exit(5)
			}
			_, _ = fmt.Fprintf(os.Stderr, "%d bytes of AvroX(N: %d / S: %d / C: %d)\n", int64(size1)+size2, nID, sID, cID)
		} else {
			var size2 int64
			size1, err := os.Stdout.Write(b)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "err: %s", err)
				os.Exit(5)

			}
			size2, err = io.Copy(os.Stdout, r)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "err: %s", err)
				os.Exit(5)
			}
			_, _ = fmt.Fprintf(os.Stderr, "%d bytes (unknown)\n", int64(size1)+size2)
		}
		os.Exit(0)
	}

	if *flagAvroX != "" {
		v := string(must.OkOne(io.ReadAll(r)))

		// Unquote the data if asked for
		if *flagUnquote {
			if len(v) > 1 && v[0:1] != `"` {
				v = `"` + v + `"`
			}
			var errUnquote error
			v, errUnquote = strconv.Unquote(v)
			if errUnquote != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Error while unquoting: %s\n", errUnquote)
				os.Exit(5)
			}
		}

		// we strip a lf if asked for
		if *flagHandleLF && len(v) > 1 && v[len(v)-1:] == "\n" {
			v = v[:len(v)-1]
		}

		//fmt.Fprintf(os.Stderr, "%q", v)
		var data any
		switch *flagAvroX {
		case "bytes":
			data = []byte(v)
		case "string":
			data = v
		case "int":
			var errAtoi error
			data, errAtoi = strconv.Atoi(v)
			if errAtoi != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Error while unquoting: %s\n", errAtoi)
				os.Exit(5)
			}
		}
		cID := avrox.CompNone
		if *flagSnappyBlock {
			cID = avrox.CompSnappy
		} else if *flagFlate {
			cID = avrox.CompFlate
		} else if *flagGZip {
			cID = avrox.CompGZip
		}

		if *flagQuote {
			fmt.Printf("%q", must.OkOne(avrox.MarshalBasic(data, cID)))
		} else {
			fmt.Printf("%s", must.OkOne(avrox.MarshalBasic(data, cID)))
		}
		os.Exit(0)
	}

	switch {
	case *flagSnappyBlock:
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
	case *flagSnappyStream:
		r = snappy.NewReader(r)
	case *flagGZip:
		r = must.OkOne(gzip.NewReader(r))
	case *flagFlate:
		r = flate.NewReader(r)
	}

	if *flagQuote {
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
	} else if *flagAvro != "" {
		schemaString := string(must.OkOne(os.ReadFile(*flagAvro)))
		schema := must.OkOne(avro.Parse(schemaString))
		dec := avro.NewDecoderForSchema(schema, r)
		var avroNative any
		must.Ok(dec.Decode(&avroNative))
		jsonData := must.OkOne(json.MarshalIndent(avroNative, "", "  "))
		fmt.Printf("%s\n", jsonData)
	} else {
		if !*flagSkipAvroX {
			// we check if we have avrox data
			b := make([]byte, 4)
			n := must.IgnoreOne(r.Read(b))
			if n == 0 {
				os.Exit(0)
			}
			if avrox.IsMagic(b) {
				nID, sID, cID, err := avrox.DecodeMagic(b)
				if err != nil {
					fmt.Printf("err: %s", err)
					os.Exit(5)
				}
				buf := bytes.Buffer{}
				buf.Write(b)
				must.OkSkipOne(buf.ReadFrom(r))
				if nID == avrox.NamespaceBasic {
					x := must.OkOne(avrox.UnmarshalBasic(buf.Bytes()))
					switch v := x.(type) {
					case string:
						if !*flagHandleLF && len(v) > 1 && v[len(v)-1:] != "\n" {
							fmt.Println(v)
						} else {
							fmt.Print(v)
						}
					case []byte:
						vv := string(v)
						if !*flagHandleLF && len(vv) > 1 && vv[len(vv)-1:] != "\n" {
							fmt.Println(vv)
						} else {
							fmt.Print(vv)
						}
					case map[string]any:
						fmt.Print(fmt.Sprintf("%#v\n", v)[23:])
					case int:
						if !*flagHandleLF {
							fmt.Println(v)
						} else {
							fmt.Print(v)
						}
					default:
						fmt.Printf("AvroXBasic(S: %d / C: %d)\n", sID, cID)
					}
				} else {
					fmt.Printf("AvroX(N: %d / S: %d / C: %d)\n", nID, sID, cID)
					os.Exit(0)
				}
			} else {
				must.OkSkipOne(os.Stdout.Write(b))
			}
		}
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
