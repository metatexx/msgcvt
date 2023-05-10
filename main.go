package main

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"

	"github.com/choria-io/fisk"

	"github.com/fxamacker/cbor/v2"
	"github.com/golang/snappy"
	"github.com/hamba/avro/v2"
	"github.com/metatexx/avrox"
	must "github.com/metatexx/mxx/mustfatal"
)

const appName = "msgcvt"

var fullVersion = "0.0.0-devel"

func main() {
	rc := run(os.Stdin, os.Args[1:])
	os.Exit(rc)
}
func run(r io.Reader, args []string) (rc int) {
	defer func() {
		rv := recover()
		if rv != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Recovered from panic: %v\n", rv)
			rc = 5
			return
		}
	}()

	app := fisk.New(appName, "msgcvt is a tool for converting data between different msg encodings.").
		Author("METATEXX GmbH (H.Raaf) <kontakt@metatexx.de>").
		Version(fullVersion)
	app.HelpFlag.Short('?')
	flagFile := app.Flag("file", "read from the given file").Short('f').ExistingFile()
	var flagData string
	var flagDataSet bool
	app.Flag("data", "read from the given string").Short('d').Action(func(_ *fisk.ParseContext) error {
		flagDataSet = true
		return nil
	}).StringVar(&flagData)
	var flagHex []byte
	app.Flag("hex", "read from the given hex bytes").Short('x').HexBytesVar(&flagHex)

	cmdTranslate := app.Command("translate", "translates from a format to a human readable output (usually indented JSON)")
	cmdTranslate.Default()
	subQuote := cmdTranslate.Command("quote", "quote output string (escapes)")
	subHex := cmdTranslate.Command("hex", "output data as hex bytes")
	subHexDump := cmdTranslate.Command("hexdump", "output data as a hex dump")
	subCBOR := cmdTranslate.Command("cbor", "output CBOR as JSON")
	subGOB := cmdTranslate.Command("gob", "output GOB as JSON (not working!)")
	subAvro := cmdTranslate.Command("avro", "avro base schema as file (will also set From to 'avro' format and To as json 'format')")
	var avroSchema string
	subAvro.Arg("file", "avro schema to use").ExistingFileVar(&avroSchema)
	subRAW := cmdTranslate.Command("raw", "no translation (but detects AvroX by default)").Default()
	flagAvroX := subRAW.Flag("avrox", "don't check for avrox in raw mode").Default("true").Bool()
	flagDecimalAsFloat := subRAW.Flag("decimal-float", "outputs AvroxBasicDecimal as float64 instead of big.Rat(io)").Default("false").UnNegatableBool()
	var flagEnsureLF bool
	subRAW.Flag("ensure-lf", "make sure the ouput ends with a linefeed").Short('l').UnNegatableBoolVar(&flagEnsureLF)

	var decompressSnappyStream bool
	app.Flag("snappy-stream", "decode data with snappy (stream mode)").UnNegatableBoolVar(&decompressSnappyStream)
	var decompressSnappyBlock bool
	app.Flag("snappy", "decode data with snappy (block mode)").UnNegatableBoolVar(&decompressSnappyBlock)
	var decompressGZip bool
	app.Flag("gzip", "decompress data with GZip").UnNegatableBoolVar(&decompressGZip)
	var decompressFlate bool
	app.Flag("deflate", "decompress data with deflate").UnNegatableBoolVar(&decompressFlate)

	cmdAnalyse := app.Command("analyse", "pipes data and gives some info about it on stderr without changing it (optionally quoting).")
	var flagQuote bool
	cmdAnalyse.Flag("quote", "quote output string (escapes)").Short('q').UnNegatableBoolVar(&flagQuote)

	cmdAvroX := app.Command("avrox", "create an AvroX basic type (string|int|bytes|decimal)")
	AvroXBasicSchema := cmdAvroX.Arg("type", "one of string,int,bytes|decimal").Required().
		Enum("string", "int", "bytes", "decimal")
	var flagUnquote bool
	cmdAvroX.Flag("unquote", "removes quotes from start and end of the data before parsing").Short('u').UnNegatableBoolVar(&flagUnquote)
	var flagStripLF bool
	cmdAvroX.Flag("strip-lf", "removes LF at the end of data before parsing").Short('s').UnNegatableBoolVar(&flagStripLF)
	var compressionType string
	cmdAvroX.Flag("compress", "set compression type for AcroX data").Short('c').
		EnumVar(&compressionType, "snappy", "gzip", "flate")

	appCmd := app.MustParseWithUsage(args)

	if flagDataSet {
		r = strings.NewReader(flagData)
	} else if len(flagHex) > 0 {
		r = bytes.NewReader(flagHex)
	} else if *flagFile != "" {
		r = must.OkOne(os.Open(*flagFile))
	}

	if flag.NArg() == 1 {
		r = strings.NewReader(flag.Arg(0))
	}

	switch {
	case decompressSnappyBlock:
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
	case decompressSnappyStream:
		r = snappy.NewReader(r)
	case decompressGZip:
		r = must.OkOne(gzip.NewReader(r))
	case decompressFlate:
		r = flate.NewReader(r)
	}

	switch appCmd {
	case cmdAnalyse.FullCommand():
		return doAnalyse(r, flagQuote)
	case cmdAvroX.FullCommand():
		return doAvroX(r, *AvroXBasicSchema, flagUnquote, flagStripLF, flagQuote, compressionType)
	case subQuote.FullCommand():
		var buf bytes.Buffer
		must.OkSkipOne(buf.ReadFrom(r))
		fmt.Printf("%q\n", buf.String())
	case subHex.FullCommand():
		var buf bytes.Buffer
		must.OkSkipOne(buf.ReadFrom(r))
		fmt.Printf("%x\n", buf.String())
	case subHexDump.FullCommand():
		stdoutDumper := hex.Dumper(os.Stdout)
		defer func() { _ = stdoutDumper.Close() }()
		must.OkSkipOne(io.Copy(stdoutDumper, r))
	case subCBOR.FullCommand():
		dec := cbor.NewDecoder(r)
		var cborNative any
		must.Ok(dec.Decode(&cborNative))
		mappedKeys := must.OkOne(convertMapKeysToStrings(cborNative))
		jsonData := must.OkOne(json.MarshalIndent(mappedKeys, "", "  "))
		fmt.Printf("%s\n", jsonData)
	case subGOB.FullCommand():
		dec := gob.NewDecoder(r)
		// Decode the gob-encoded map
		var gobNative map[string]interface{}
		must.Ok(dec.Decode(&gobNative))
		// Convert the decoded data to JSON
		jsonData := must.OkOne(json.MarshalIndent(gobNative, "", "  "))
		fmt.Printf("%s\n", jsonData)
	case subAvro.FullCommand():
		schemaString := string(must.OkOne(os.ReadFile(avroSchema)))
		schema := must.OkOne(avro.Parse(schemaString))
		dec := avro.NewDecoderForSchema(schema, r)
		var avroNative any
		must.Ok(dec.Decode(&avroNative))
		jsonData := must.OkOne(json.MarshalIndent(avroNative, "", "  "))
		fmt.Printf("%s\n", jsonData)
	case subRAW.FullCommand():
		if *flagAvroX {
			// we check if we have avrox data
			b := make([]byte, avrox.MagicLen)
			n := must.IgnoreOne(r.Read(b))
			if n == 0 {
				return 0
			}
			b = b[:n]
			if avrox.IsMagic(b) {
				nID, sID, cID, err := avrox.DecodeMagic(b)
				if err != nil {
					fmt.Printf("err: %s", err)
					return 5
				}
				buf := bytes.Buffer{}
				buf.Write(b)
				must.OkSkipOne(buf.ReadFrom(r))
				if nID == avrox.NamespaceBasic {
					x := must.OkOne(avrox.UnmarshalBasic(buf.Bytes()))
					switch v := x.(type) {
					case string:
						if flagEnsureLF && len(v) > 1 && v[len(v)-1:] != "\n" {
							fmt.Println(v)
						} else {
							fmt.Print(v)
						}
					case []byte:
						vv := string(v)
						if flagEnsureLF && len(vv) > 1 && vv[len(vv)-1:] != "\n" {
							fmt.Println(vv)
						} else {
							fmt.Print(vv)
						}
					case map[string]any:
						fmt.Print(fmt.Sprintf("%#v\n", v)[23:])
					case *big.Rat:
						var out any
						if *flagDecimalAsFloat {
							out, _ = v.Float64()
						} else {
							out = v.String()
						}
						if flagEnsureLF {
							fmt.Println(out)
						} else {
							fmt.Print(out)
						}
					case int:
						if flagEnsureLF {
							fmt.Println(v)
						} else {
							fmt.Print(v)
						}
					default:
						fmt.Printf("AvroXBasic(S: %d / C: %d)\n", sID, cID)
					}
					return 0
				} else {
					fmt.Printf("AvroX(N: %d / S: %d / C: %d)\n", nID, sID, cID)
					return 0
				}
			} else {
				must.OkSkipOne(os.Stdout.Write(b))
			}
		}
		must.OkSkipOne(io.Copy(os.Stdout, r))
		if flagEnsureLF {
			// ToDo: Actually test for the last rune being a lf
			fmt.Println()
		}
		//fmt.Println("unsupported format conversion")
	}
	return 0
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
