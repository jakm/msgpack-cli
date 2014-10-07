#!/bin/bash

clean_test() {
    [ "$RPC_SERVER_PID" != "" ] && kill $RPC_SERVER_PID 2> /dev/null
    [ "$TESTDIR" != "" ] && rm -rf $TESTDIR
}

fail() {
    clean_test
    echo
    echo -e "\e[31mFAILED: ${1}\e[39m"
    exit 1
}

cancel() {
    fail "CTRL-C detected"
}

info() {
    echo -e "\e[34m${1}\e[39m"
    echo
}

trap cancel INT

# -------------------

info "Preparing test environment..."

TESTDIR=$(mktemp -d)

virtualenv $TESTDIR > /dev/null || fail "virtualenv creation"

cd $(dirname $0)/..
go build -o $TESTDIR/bin/msgpack-cli || fail "msgpack-cli build"

cp test/assert-json-equal $TESTDIR/bin
cp test/assert-msgpack-equal $TESTDIR/bin
cp test/rpc-server $TESTDIR/bin

cp test/data.json $TESTDIR
cp test/data.bin $TESTDIR

cd $TESTDIR

. $TESTDIR/bin/activate

pip install msgpack-rpc-python > /dev/null || fail "msgpack-rpc-python installation"

echo "Starting rpc-server on localhost:8000..."
echo

./bin/rpc-server &

RPC_SERVER_PID=$(echo $!)

sleep 1

if ! kill -0 $RPC_SERVER_PID; then
    fail "rpc-server is not running"
fi

# -------------------

info "Testing RPC..."

echo "RPC: echo"
assert-json-equal "$(msgpack-cli rpc localhost 8000 echo)" "[]"

echo "RPC: echo 3.14159"
assert-json-equal "$(msgpack-cli rpc localhost 8000 echo 3.14159)" "[3.14159]"

echo "RPC: echo test"
assert-json-equal "$(msgpack-cli rpc localhost 8000 echo text)" '["text"]'

echo "RPC: echo \"long test\""
assert-json-equal "$(msgpack-cli rpc localhost 8000 echo "long text")" '["long text"]'

echo "RPC: echo '[\"abc\", \"def\", \"ghi\", {\"A\": 65, \"B\": 66, \"C\": 67}]'"
assert-json-equal "$(msgpack-cli rpc localhost 8000 echo '["abc", "def", "ghi", {"A": 65, "B": 66, "C": 67}]')" '["abc","def","ghi",{"A":65,"B":66,"C":67}]'

# -------------------

info "Testing encoding/decoding..."

echo "Encode: data.json"
msgpack-cli encode data.json --out=output.bin
assert-msgpack-equal output.bin data.bin

echo "Decode: data.bin"
assert-json-equal "$(msgpack-cli decode data.bin)" "$(cat data.json)"

# -------------------

clean_test
