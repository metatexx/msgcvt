#!/bin/bash

# This was only tested on Darwin

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
# Change the working directory to the directory of the script
cd "$SCRIPT_DIR" || exit 5

echo -n "testing avrox basic string 1: "
test "$(msgcvt -avrox string "test" | msgcvt -n | msgcvt -avrox string | msgcvt | xxd)" = \
 "$(echo -e -n "test\n" | xxd)" && echo "ok" || echo "failed"
echo -n "testing avrox basic string 2: "
test "$(msgcvt -e -avrox string "test\n" | msgcvt -n | msgcvt -n -avrox string | msgcvt | xxd)" = \
 "$(echo -e -n "test\n" | xxd)" && echo "ok" || echo "failed"
echo -n "testing avrox basic int: "
test "$(msgcvt -avrox int 42 | msgcvt -n | msgcvt -avrox int | msgcvt)" = 42 && echo "ok" || echo "failed"
echo -n "testing avrox basic string from file: "
test "$(msgcvt -avrox string -f test.txt | msgcvt -n | msgcvt -avrox string | msgcvt | xxd)" = \
 "$(echo -e -n "Wello Horld!\n" | xxd)" && echo "ok" || echo "failed"
echo -n "testing avrox basic string from file (snappy): "
test "$(msgcvt -avrox string -f test.txt | msgcvt -n | msgcvt -avrox string -snappy | msgcvt | xxd)" = \
 "$(echo -e -n "Wello Horld!\n" | xxd)" && echo "ok" || echo "failed"
echo -n "testing avrox basic string from file (flate): "
test "$(msgcvt -avrox string -f test.txt | msgcvt -n | msgcvt -avrox string -flate | msgcvt | xxd)" = \
 "$(echo -e -n "Wello Horld!\n" | xxd)" && echo "ok" || echo "failed"
echo -n "testing avrox basic string from file (gzip): "
test "$(msgcvt -avrox string -f test.txt | msgcvt -n | msgcvt -avrox string -gzip | msgcvt | xxd)" = \
 "$(echo -e -n "Wello Horld!\n" | xxd)" && echo "ok" || echo "failed"
echo -n "file to snappy encoded avrox bytes and back: "
test "$(msgcvt -f /bin/sh -snappy -avrox bytes | msgcvt -n | xxd)" = \
 "$(cat /bin/sh | xxd)" && echo "ok" || echo "failed"
