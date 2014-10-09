msgpack-cli
===========

msgpack-cli is command line tool that converts data from JSON to [Msgpack](http://msgpack.org) and vice versa. Also allows calling RPC methods via [msgpack-rpc](https://github.com/msgpack-rpc/msgpack-rpc/blob/master/spec.md).

Installation
------------

    go get github.com/jakm/msgpack-cli

Usage
-----

    msgpack-cli

    Usage:
        msgpack-cli encode <input-file> [--out=<output-file>]
        msgpack-cli decode <input-file> [--out=<output-file>]
        msgpack-cli rpc <host> <port> <method> [<params>|--file=<input-file>]
        msgpack-cli -h | --help
        msgpack-cli --version

    Commands:
        encode               Encode data from input file to STDOUT
        decode               Decode data from input file to STDOUT
        rpc                  Call RPC method and write result to STDOUT

    Options:
        -h --help            Show this help message and exit
        --version            Show version
        --out=<output-file>  Write output data to file instead of STDOUT
        --file=<input-file>  File where parameters or RPC method are read from

    Arguments:
        <input-file>         File where data are read from
        <host>               Server hostname
        <port>               Server port
        <method>             Name of RPC method
        <params>             Parameters of RPC method in JSON format

Examples
--------

Encoding/decoding:

    $ cat test.json
    {
      "firstName": "John",
      "lastName": "Smith",
      "isAlive": true,
      "age": 25,
      "height_cm": 167.6,
      "address": {
        "streetAddress": "21 2nd Street",
        "city": "New York",
        "state": "NY",
        "postalCode": "10021-3100"
      },
      "phoneNumbers": [
        {
          "type": "home",
          "number": "212 555-1234"
        },
        {
          "type": "office",
          "number": "646 555-4567"
        }
      ],
      "children": [],
      "spouse": null
    }
    $
    $ msgpack-cli encode test.json --out test.bin
    $
    $ ls -l test.* | awk '{print $9, $5}'
    test.bin 250
    test.json 429
    $
    $ msgpack-cli decode test.bin
    {"address":{"city":"New York","postalCode":"10021-3100","state":"NY","streetAddress":"21 2nd Street"},"age":25,"children":[],"firstName":"John","height_cm":167.6,"isAlive":true,"lastName":"Smith","phoneNumbers":[{"number":"212 555-1234","type":"home"},{"number":"646 555-4567","type":"office"}],"spouse":null}

RPC calling:

    $ # zero params
    $ msgpack-cli rpc localhost 8000 echo
    []
    $
    $ # single param
    $ msgpack-cli rpc localhost 8000 echo 3.159
    [3.159]
    $
    $ # multiple params (as json array)
    $ msgpack-cli rpc localhost 8000 echo '["abc", "def", "ghi", {"A": 65, "B": 66, "C": 67}]'
    ["abc","def","ghi",{"A":65,"B":66,"C":67}]

