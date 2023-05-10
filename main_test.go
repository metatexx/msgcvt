package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func Test_run(t *testing.T) {
	type args struct {
		reader io.Reader
		args   []string
	}
	tests := []struct {
		name       string
		args       args
		wantOutput []byte
		wantRc     int
	}{
		{"simple", args{strings.NewReader("test"), []string{}}, []byte("test"), 0},
		{"empty", args{strings.NewReader(""), []string{}}, []byte(""), 0},
		{"zero", args{bytes.NewReader([]byte{0}), []string{}}, []byte{0}, 0},
		{"avrox-string", args{strings.NewReader("test\n"), []string{"avrox", "string"}},
			append([]byte{147, 0, 0, 1, 0, 0, 1, 254, 10}, []byte("test\n")...), 0},
		{"avrox-decimal", args{strings.NewReader("1.3\n"), []string{"avrox", "decimal"}},
			[]byte("\x93\x00\x00\x01\x00\x00\x06\xf9\x02\x042\xc8"), 0},
		{"strip-lf", args{strings.NewReader("test\n"), []string{"avrox", "-s", "string"}},
			[]byte("\x93\x00\x00\x01\x00\x00\x01\xfe\btest"), 0},
		{"unquote", args{strings.NewReader(`test\n`), []string{"avrox", "-u", "string"}},
			[]byte("\x93\x00\x00\x01\x00\x00\x01\xfe\ntest\n"), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalStdout := os.Stdout
			// Create a new pipe and make it the stdout.
			r, w, _ := os.Pipe()
			os.Stdout = w
			if gotRc := run(tt.args.reader, tt.args.args); gotRc != tt.wantRc {
				t.Errorf("run() = %v, want %v", gotRc, tt.wantRc)
			}
			// Restore the original stdout.
			os.Stdout = originalStdout

			// Close the writer and read what was written.
			_ = w.Close()
			var buf bytes.Buffer
			_, _ = buf.ReadFrom(r)
			gotOutput := buf.Bytes()
			// Check the output.
			if !bytes.Equal(gotOutput, tt.wantOutput) {
				fmt.Printf("%q, want %q", string(gotOutput), string(tt.wantOutput))
				t.Errorf("run() = %v, want %v", gotOutput, tt.wantOutput)

			}
		})
	}
}
