// Copyright 2014-2015 Jakub Matys
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
    "fmt"
    "github.com/docopt/docopt-go"
    "io/ioutil"
    "log"
    "strconv"
)

const usage = `msgpack-cli

Usage:
    msgpack-cli encode <input-file> [--out=<output-file>] [--disable-int64-conv]
    msgpack-cli decode <input-file> [--out=<output-file>] [--pp]
    msgpack-cli rpc <host> <port> <method> [<params>|--file=<input-file>] [--pp]
        [--timeout=<timeout>][--disable-int64-conv]
    msgpack-cli -h | --help
    msgpack-cli --version

Commands:
    encode                Encode data from input file to STDOUT
    decode                Decode data from input file to STDOUT
    rpc                   Call RPC method and write result to STDOUT

Options:
    -h --help             Show this help message and exit
    --version             Show version
    --out=<output-file>   Write output data to file instead of STDOUT
    --file=<input-file>   File where parameters or RPC method are read from
    --pp                  Pretty-print - indent output JSON data
    --timeout=<timeout>   Timeout of RPC call [default: 30]
    --disable-int64-conv  Disable the default behaviour such that JSON numbers
                          are converted to float64 or int64 numbers by their
                          meaning, all result numbers will have float64 type


Arguments:
    <input-file>          File where data are read from
    <host>                Server hostname
    <port>                Server port
    <method>              Name of RPC method
    <params>              Parameters of RPC method in JSON format`

type Options struct {
    convertToInt64 bool
    indent         bool
    timeout        uint32
}

func main() {
    arguments, err := docopt.Parse(usage, nil, true, "msgpack-cli "+__VERSION__, false)
    if err != nil {
        log.Fatal(fmt.Errorf("Arguments parsing: %s", err))
    }

    switch {
    case arguments["encode"], arguments["decode"]:
        inFilename := arguments["<input-file>"].(string)
        outFilename, _ := arguments["--out"].(string)

        conversionFunc := ConvertJSON2Msgpack
        if arguments["decode"].(bool) {
            conversionFunc = ConvertMsgpack2JSON
        }

        options := Options{
            convertToInt64: !arguments["--disable-int64-conv"].(bool),
            indent:         arguments["--pp"].(bool),
        }

        err = ConvertFormats(inFilename, outFilename, conversionFunc, options)
    case arguments["rpc"]:
        host := arguments["<host>"].(string)
        port := arguments["<port>"].(string)
        method := arguments["<method>"].(string)
        var params string
        params, err = getRPCParams(arguments)
        if err != nil {
            break
        }
        var timeout uint32
        timeout, err = getTimeout(arguments)
        if err != nil {
            break
        }

        options := Options{
            convertToInt64: !arguments["--disable-int64-conv"].(bool),
            indent:         arguments["--pp"].(bool),
            timeout:        timeout,
        }

        err = CallRPC(host, port, method, params, options)
    default:
        panic("unreachable")
    }
    if err != nil {
        log.Fatal(err)
    }
}

func getRPCParams(arguments map[string]interface{}) (params string, err error) {
    params, _ = arguments["<params>"].(string)
    filename, _ := arguments["--file"].(string)

    if filename != "" {
        buff, err := ioutil.ReadFile(filename)
        if err != nil {
            return "", err
        }
        params = string(buff)
    }

    return params, nil
}

func getTimeout(arguments map[string]interface{}) (timeout uint32, err error) {
    timeout = uint32(30)
    if str := arguments["--timeout"].(string); str != "" {
        var tmp uint64
        tmp, err = strconv.ParseUint(str, 10, 32)
        timeout = uint32(tmp)
    }
    return timeout, err
}
