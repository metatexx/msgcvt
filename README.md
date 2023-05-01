# msgcvt - use it with `nats --translate`

## Introduction

`msgcvt` is a command-line utility designed to convert data between different message encodings. This tool offers a variety of functionalities including translation of formats to human-readable output, analysis of data, and creation of AvroX basic types.

## Available Commands

```
usage: msgcvt [<flags>] <command> [<args> ...]

msgcvt is a tool for converting data between different msg encodings.

Commands:
  translate  translates from a format to a human readable output (usually indented JSON)
  analyse    pipes data and gives some info about it on stderr without changing it (optionally quoting).
  avrox      create an AvroX basic type (string|int|bytes)

Global Flags:
  -?, --help           Show context-sensitive help
      --version        Show application version.
  -f, --file=FILE      read from the given file
  -d, --data=DATA      read from the given string
  -x, --hex=HEX        read from the given hex bytes
      --snappy-stream  decode data with snappy (stream mode)
      --snappy         decode data with snappy (block mode)
      --gzip           decompress data with GZip
      --deflate        decompress data with deflate
```

## Conclusion

`msgcvt` is a versatile utility that simplifies the process of working with different message encodings. Whether you need to translate, analyse data, or create AvroX basic types, `msgcvt` has got you covered. For any additional help or usage examples, please use the `-?, --help` flag. 

Happy data converting!