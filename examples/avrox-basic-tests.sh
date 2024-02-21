#!/bin/bash

# This was only tested on Darwin

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
# Change the working directory to the directory of the script
cd "$SCRIPT_DIR" || exit 5

echo -n "testing avrox basic string 1: "
test "$(msgcvt avrox string -d "test" | msgcvt | msgcvt avrox string | msgcvt | xxd)" = \
 "$(echo -e -n "test" | xxd)" && echo "ok" || echo "failed"
echo -n "testing avrox basic string 2: "
test "$(echo "test" | msgcvt avrox -s string | msgcvt | xxd)" = \
 "$(echo -n "test" | xxd)" && echo "ok" || echo "failed"
echo -n "testing avrox basic int: "
test "$(msgcvt avrox int -d 42 | msgcvt | msgcvt avrox int | msgcvt)" = 42 && echo "ok" || echo "failed"
echo -n "testing avrox basic decimal: "
test "$(msgcvt avrox decimal -d 1.3 | msgcvt --decimal-float)" = "1.3" && echo "ok" || echo "failed"
echo -n "testing avrox basic string from file: "
test "$(msgcvt avrox string -f test.txt | msgcvt | msgcvt avrox string | msgcvt | xxd)" = \
 "$(echo -e -n "Wello Horld!\n" | xxd)" && echo "ok" || echo "failed"
echo -n "testing avrox basic string from file (snappy): "
test "$(msgcvt avrox string -f test.txt | msgcvt | msgcvt avrox string -c snappy | msgcvt | xxd)" = \
 "$(echo -e -n "Wello Horld!\n" | xxd)" && echo "ok" || echo "failed"
echo -n "testing avrox basic string from file (flate): "
test "$(msgcvt avrox string -f test.txt | msgcvt | msgcvt avrox string -c flate | msgcvt | xxd)" = \
 "$(echo -e -n "Wello Horld!\n" | xxd)" && echo "ok" || echo "failed"
echo -n "testing avrox basic string from file (gzip): "
test "$(msgcvt avrox string -f test.txt | msgcvt | msgcvt avrox string -c gzip | msgcvt | xxd)" = \
 "$(echo -e -n "Wello Horld!\n" | xxd)" && echo "ok" || echo "failed"
echo -n "file to snappy encoded avrox bytes and back: "
test "$(msgcvt -f /bin/sh avrox bytes -c snappy | msgcvt | xxd)" = \
 "$(cat /bin/sh | xxd)" && echo "ok" || echo "failed"
 echo -n "rawdate from a string representation: "
test "$(msgcvt avrox rawdate -d "2024-02-20" | msgcvt | msgcvt avrox rawdate -c gzip | msgcvt | xxd)" = \
 "$(echo -e -n "2024-02-20" | xxd)" && echo "ok" || echo "failed"
