package main

import (
	"fmt"
	"github.com/metatexx/avrox"
	must "github.com/metatexx/mxx/mustfatal"
	"io"
	"os"
)

func doAnalyse(r io.Reader, flagQuote bool) int {
	b := make([]byte, 4)
	n := must.IgnoreOne(r.Read(b))
	if n == 0 {
		_, _ = fmt.Fprintf(os.Stderr, "0 bytes\n")
		return 0
	}
	if avrox.IsMagic(b) {
		nID, sID, cID, err := avrox.DecodeMagic(b)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "err: %s", err)
			return 5
		}
		var size1 int
		var size2 int64
		if flagQuote {
			size1 = len(b)
			b2, _ := io.ReadAll(r)
			size2 = int64(len(b2))
			fmt.Printf("%q", string(b)+string(b2))
		} else {
			size1, err = os.Stdout.Write(b)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "err: %s", err)
				return 5

			}
			size2, err = io.Copy(os.Stdout, r)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "err: %s", err)
				return 5
			}
		}
		_, _ = fmt.Fprintf(os.Stderr, "%d bytes of AvroX(N: %d / S: %d / C: %d)\n", int64(size1)+size2, nID, sID, cID)
	} else {
		var size2 int64
		size1, err := os.Stdout.Write(b)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "err: %s", err)
			return 5

		}
		size2, err = io.Copy(os.Stdout, r)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "err: %s", err)
			return 5
		}
		_, _ = fmt.Fprintf(os.Stderr, "%d bytes (unknown)\n", int64(size1)+size2)
	}
	return 0
}
