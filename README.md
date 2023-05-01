# msgcvt - use it with `nats --translate`

## Introduction

`msgcvt` is a command-line utility designed to convert data between different message encodings. This tool offers a variety of functionalities including translation of formats to human-readable output, analysis of data, and creation of AvroX basic types.

## Available Commands

```
usage: msgcvt [<flags>] <command> [<args> ...]

msgcvt is a tool for converting data between different msg encodings.

Flags:
  -?, --help           Show context-sensitive help
      --version        Show application version.
  -f, --file=FILE      read from the given file
  -d, --data=DATA      read from the given string
  -x, --hex=HEX        read from the given hex bytes
      --snappy-stream  decode data with snappy (stream mode)
      --snappy         decode data with snappy (block mode)
      --gzip           decompress data with GZip
      --deflate        decompress data with deflate
      --[no-]avrox     don't check for avrox in raw mode
  -l, --ensure-lf      make sure the ouput ends with a linefeed

Commands:
help [<command>...]
    Show help.


translate quote
    quote output string (escapes)


translate hex
    output data as hex bytes


translate hexdump
    output data as a hex dump


translate cbor
    output CBOR as JSON


translate gob
    output GOB as JSON (not working!)


translate avro [<file>]
    avro base schema as file (will also set From to 'avro' format and To as json 'format')


translate raw [<flags>]
    no translation (but detects AvroX by default)

        --[no-]avrox  don't check for avrox in raw mode
    -l, --ensure-lf   make sure the ouput ends with a linefeed

analyse [<flags>]
    pipes data and gives some info about it on stderr without changing it (optionally quoting).

    -q, --quote  quote output string (escapes)

avrox [<flags>] <type>
    create an AvroX basic type (string|int|bytes)

    -u, --unquote            removes quotes from start and end of the data before parsing
    -s, --strip-lf           removes LF at the end of data before parsing
    -c, --compress=COMPRESS  set compression type for AcroX data
```

## Conclusion

`msgcvt` is a versatile utility that simplifies the process of working with different message encodings. Whether you need to translate, analyse data, or create AvroX basic types, `msgcvt` has got you covered. For any additional help or usage examples, please use the `-?, --help` flag. 

Happy data converting!