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
	"github.com/metatexx/avrox/rawdate"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

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

var flagAvroX bool
var avscPaths []string
var verboseAVSC bool
var noBasicsDetection bool
var decimalAsFloat bool

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
	cmdTranslate.Default().Alias("t")
	subQuote := cmdTranslate.Command("quote", "quote output string (escapes)").Default().Alias("q")
	addAvroXFlags(subQuote)
	subRAW := cmdTranslate.Command("raw", "no translation (but detects AvroX by default)").Alias("r")
	addAvroXFlags(subRAW)
	var flagEnsureLF bool
	subRAW.Flag("ensure-lf", "make sure the ouput ends with a linefeed").Short('l').UnNegatableBoolVar(&flagEnsureLF)
	subHex := cmdTranslate.Command("hex", "output data as hex bytes").Alias("h")
	addAvroXFlags(subHex)
	subHexDump := cmdTranslate.Command("hexdump", "output data as a hex dump").Alias("d")
	subCBOR := cmdTranslate.Command("cbor", "output CBOR as JSON")
	subGOB := cmdTranslate.Command("gob", "output GOB as JSON (not working!)")
	subAvro := cmdTranslate.Command("avro", "avro base schema as file (will also set From to 'avro' format and To as json 'format')")
	var avroSchema string
	subAvro.Arg("file", "avro schema to use").ExistingFileVar(&avroSchema)

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
		Enum("string", "int", "bytes", "decimal", "rawdate")
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
	case subQuote.FullCommand():
		fallthrough
	case subHex.FullCommand():
		fallthrough
	case subRAW.FullCommand():
		buf := bytes.Buffer{}
		if flagAvroX {
			// scan for avsc files if requested
			avroxSchemas := map[string]avro.NamedSchema{}
			if len(avscPaths) > 0 {
				// TODO: we should implement some system wide cache for that
				err := scanForAVSC(avscPaths, avroxSchemas, verboseAVSC)
				app.FatalIfError(err, "scanning avsc")
			}
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
				if !noBasicsDetection && nID == avrox.NamespaceBasic {
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
						if decimalAsFloat {
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
					case time.Time:
						if flagEnsureLF {
							fmt.Println(v.Format(time.RFC3339))
						} else {
							fmt.Print(v.Format(time.RFC3339))
						}
					case rawdate.RawDate:
						if flagEnsureLF {
							fmt.Println(v.String())
						} else {
							fmt.Print(v.String())
						}
					default:
						fmt.Printf("UnknownAvroXBasic(S: %d / C: %d)\n", sID, cID)
					}
					return 0
				} else {
					avroxID := fmt.Sprintf("%d.%d.%d", nID, sID>>8, sID&0xff)
					fmt.Printf("AvroX(%d.%d.%d / C: %d / L: %d)\n", nID, sID>>8, sID&0xff, cID, buf.Len())
					if schema, found := avroxSchemas[avroxID]; found {
						var avroNative map[string]any
						_, _, err = avrox.UnmarshalAny(buf.Bytes(), schema, &avroNative)
						app.FatalIfError(err, "can't unmarshal data")
						// change the byte array to a readable string
						avroNative["Magic"] = fmt.Sprintf("%x", avroNative["Magic"])
						jsonData := must.OkOne(json.MarshalIndent(avroNative, "", "  "))
						fmt.Printf("%s\n", jsonData)
					}
					return 0
				}
			} else if appCmd == subHex.FullCommand() {
				buf.Write(b)
			} else if appCmd == subQuote.FullCommand() {
				buf.Write(b)
			} else {
				must.OkSkipOne(os.Stdout.Write(b))
			}
		}
		if appCmd == subHex.FullCommand() {
			must.OkSkipOne(buf.ReadFrom(r))
			fmt.Printf("%x\n", buf.String())
		} else if appCmd == subQuote.FullCommand() {
			must.OkSkipOne(buf.ReadFrom(r))
			fmt.Printf("%q\n", buf.String())
		} else {
			must.OkSkipOne(io.Copy(os.Stdout, r))
			if flagEnsureLF {
				// ToDo: Actually test for the last rune being a lf
				fmt.Println()
			}
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

func scanForAVSC(paths []string, schemas map[string]avro.NamedSchema, verbose bool) error {
	for _, path := range paths {
		err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				base := filepath.Base(path)
				if strings.HasSuffix(base, "_tests") {
					return filepath.SkipDir
				}
				switch base {
				case ".git", ".idea":
					return filepath.SkipDir
				}
			}
			if !info.IsDir() && strings.HasSuffix(info.Name(), ".avsc") {
				schema, err := avro.ParseFiles(path)
				if err != nil {
					if verbose {
						_, _ = fmt.Fprintf(os.Stderr, "scanning path %q: %v\n", path, err)
					}
					return nil
				}
				var nSchema avro.NamedSchema
				nSchema = schema.(avro.NamedSchema)
				tmp := nSchema.Prop("avrox")
				if avroXID, ok := tmp.(string); ok {
					if verbose {
						_, _ = fmt.Fprintf(os.Stderr, "%s (%s)\n", nSchema.Name(), avroXID)
					}
					schemas[avroXID] = nSchema
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func addAvroXFlags(cmd *fisk.CmdClause) {
	cmd.Flag("avrox", "don't check for avrox in raw mode").Default("true").BoolVar(&flagAvroX)
	flagAVSC := cmd.Flag("avsc", "paths that are recursively scanned for files with an '.avsc' extension. If found, they get parsed and being used to decode corresponding AvroX data")
	flagAVSC.ExistingFilesOrDirsVar(&avscPaths)
	cmd.Flag("verbose", "Gives information about the avsc scanning phase (incl. the otherwise suppressed errors)").
		Short('v').UnNegatableBoolVar(&verboseAVSC)
	cmd.Flag("no-basics", "No special AvroX basics detection (decodes them like other avrox data using the avsc schemas)").
		Short('b').UnNegatableBoolVar(&noBasicsDetection)
	cmd.Flag("decimal-float", "outputs AvroxBasicDecimal as float64 instead of big.Rat(io)").Default("false").UnNegatableBoolVar(&decimalAsFloat)
}
