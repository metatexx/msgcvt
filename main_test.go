package main

import (
	"bytes"
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
			append([]byte{147, 1, 0, 9, 10}, []byte("test\n")...), 0},
		{"strip-lf", args{strings.NewReader("test\n"), []string{"avrox", "-s", "string"}},
			append([]byte{147, 1, 0, 9, 8}, []byte("test")...), 0},
		{"unquote", args{strings.NewReader(`test\n`), []string{"avrox", "-u", "string"}},
			append([]byte{147, 1, 0, 9, 10}, []byte("test\n")...), 0},
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
				t.Errorf("run() = %v, want %v", gotOutput, tt.wantOutput)
			}
		})
	}
}