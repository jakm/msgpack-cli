// Copyright 2014 Jakub Matys
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
    "encoding/json"
    "fmt"
    "github.com/docopt/docopt-go"
    "github.com/ugorji/go/codec"
    "io/ioutil"
    "log"
    "net"
    "net/rpc"
    "os"
    "reflect"
    "strconv"
    "strings"
    "unicode"
    "unicode/utf8"
)

const usage = `msgpack-cli

Usage:
    msgpack-cli encode <input-file> [--out=<output-file>]
    msgpack-cli decode <input-file> [--out=<output-file>] [--pp]
    msgpack-cli rpc <host> <port> <method> [<params>|--file=<input-file>] [--pp]
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
    --pp                 Pretty-print - indent output JSON data

Arguments:
    <input-file>         File where data are read from
    <host>               Server hostname
    <port>               Server port
    <method>             Name of RPC method
    <params>             Parameters of RPC method in JSON format`

type Options struct {
    indent bool
}

func main() {
    arguments, err := docopt.Parse(usage, nil, true, "msgpack-cli "+__VERSION__, false)
    if err != nil {
        log.Fatal(err)
    }

    switch {
    case arguments["encode"], arguments["decode"]:
        inFilename := arguments["<input-file>"].(string)
        outFilename, _ := arguments["--out"].(string)

        convertFunc := convertJSON2Msgpack
        if arguments["decode"].(bool) {
            convertFunc = convertMsgpack2JSON
        }

        options := Options{indent: arguments["--pp"].(bool)}

        err = doConversion(inFilename, outFilename, convertFunc, options)
    case arguments["rpc"]:
        host := arguments["<host>"].(string)
        port := arguments["<port>"].(string)
        method := arguments["<method>"].(string)
        params, err := getRPCParams(arguments)
        if err != nil {
            break
        }

        options := Options{indent: arguments["--pp"].(bool)}

        err = doRPC(host, port, method, params, options)
    default:
        panic("unreachable")
    }
    if err != nil {
        log.Fatal(err)
    }
}

func doConversion(inFilename, outFilename string, convertFunc func(data []byte, options Options) ([]byte, error), options Options) error {
    inFile, err := os.Open(inFilename)
    if err != nil {
        return err
    }
    defer inFile.Close()

    outFile := os.Stdout
    if outFilename != "" {
        if outFile, err = os.Create(outFilename); err != nil {
            return err
        }
        defer outFile.Close()
    }

    var inBuffer, outBuffer []byte

    if inBuffer, err = ioutil.ReadAll(inFile); err != nil {
        return fmt.Errorf("Reading error: %s", err)
    }

    if outBuffer, err = convertFunc(inBuffer, options); err != nil {
        return err
    }

    var n int

    if n, err = outFile.Write(outBuffer); err != nil {
        return fmt.Errorf("Writing error: %s", err)
    }
    if n != len(outBuffer) {
        return fmt.Errorf("Writing error: written %d of %d bytes", n, len(outBuffer))
    }

    return nil
}

func doRPC(host, port, method, params string, options Options) error {
    if len(params) == 0 {
        params = "[]"
    } else if !strings.HasPrefix(params, "[") {
        if char, _ := utf8.DecodeRuneInString(params); unicode.IsLetter(char) {
            params = strconv.Quote(params)
        }
        params = "[" + params + "]"
    }

    args, err := decodeJSON(params)
    if err != nil {
        return err
    }

    var conn net.Conn
    if conn, err = net.Dial("tcp", host+":"+port); err != nil {
        return err
    }
    defer conn.Close()

    var reply interface{}
    if reply, err = callRPC(conn, method, args); err != nil {
        return err
    }

    var jsonData string
    if jsonData, err = encodeJSON(reply, options.indent); err != nil {
        return err
    }

    fmt.Println(jsonData)

    return nil
}

func callRPC(conn net.Conn, method string, args interface{}) (interface{}, error) {
    handle := getHandle()
    rpcCodec := codec.MsgpackSpecRpc.ClientCodec(conn, &handle)
    client := rpc.NewClientWithCodec(rpcCodec)

    var reply interface{}
    var mArgs codec.MsgpackSpecRpcMultiArgs = args.([]interface{})

    if err := client.Call(method, mArgs, &reply); err != nil {
        return nil, fmt.Errorf("RPC error: %s", err)
    }

    return reply, nil
}

func convertJSON2Msgpack(data []byte, options Options) (result []byte, err error) {
    var object interface{}

    if object, err = decodeJSON(string(data)); err != nil {
        return nil, err
    }

    if result, err = encodeMsgpack(object); err != nil {
        return nil, err
    }

    return result, nil
}

func convertMsgpack2JSON(data []byte, options Options) (result []byte, err error) {
    var object interface{}

    if object, err = decodeMsgpack(data); err != nil {
        return nil, err
    }

    var jsonData string
    if jsonData, err = encodeJSON(object, options.indent); err != nil {
        return nil, err
    }
    result = []byte(jsonData)

    return result, nil
}

func getHandle() (handle codec.MsgpackHandle) {
    handle = codec.MsgpackHandle{RawToString: true}
    handle.MapType = reflect.TypeOf(map[string]interface{}(nil))
    return handle
}

func encodeJSON(object interface{}, indent bool) (data string, err error) {
    var bytes []byte
    if indent {
        bytes, err = json.MarshalIndent(object, "", "  ")
    } else {
        bytes, err = json.Marshal(object)
    }
    if err != nil {
        return "", fmt.Errorf("JSON encoding: %s", err)
    }
    return string(bytes), nil
}

func decodeJSON(data string) (object interface{}, err error) {
    bytes := []byte(data)
    if err := json.Unmarshal(bytes, &object); err != nil {
        return nil, fmt.Errorf("JSON decoding: %s", err)
    }
    return object, nil
}

func encodeMsgpack(object interface{}) (data []byte, err error) {
    handle := getHandle()
    encoder := codec.NewEncoderBytes(&data, &handle)
    if err := encoder.Encode(object); err != nil {
        return nil, fmt.Errorf("Msgpack encoding: %s", err)
    }
    return data, err
}

func decodeMsgpack(data []byte) (object interface{}, err error) {
    handle := getHandle()
    decoder := codec.NewDecoderBytes(data, &handle)
    if err := decoder.Decode(&object); err != nil {
        return nil, fmt.Errorf("Msgpack decoding: %s", err)
    }
    return object, err
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
