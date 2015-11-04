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
    "bytes"
    "fmt"
    "github.com/docopt/docopt-go"
    "io"
    "io/ioutil"
    "log"
    "net"
    "os"
    "strconv"
    "strings"
    "time"
    "unicode"
    "unicode/utf8"
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

type ConversionFunc func(r io.Reader, w io.Writer, options Options) error

type Options struct {
    convertToInt64 bool
    indent         bool
    timeout        uint32
}

type RPCResult struct {
    reply interface{}
    err   error
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

        conversionFunc := convertJSON2Msgpack
        if arguments["decode"].(bool) {
            conversionFunc = convertMsgpack2JSON
        }

        options := Options{
            convertToInt64: !arguments["--disable-int64-conv"].(bool),
            indent:         arguments["--pp"].(bool),
        }

        err = doConversion(inFilename, outFilename, conversionFunc, options)
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

        err = doRPC(host, port, method, params, options)
    default:
        panic("unreachable")
    }
    if err != nil {
        log.Fatal(err)
    }
}

func doConversion(inFilename, outFilename string, conversionFunc ConversionFunc, options Options) error {
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

    if err = conversionFunc(inFile, outFile, options); err != nil {
        return err
    }

    return nil
}

func adjustRPCParams(params string) string {
    if len(params) == 0 {
        params = "[]"
    } else if !strings.HasPrefix(params, "[") {
        if char, _ := utf8.DecodeRuneInString(params); unicode.IsLetter(char) {
            params = strconv.Quote(params)
        }
        params = "[" + params + "]"
    }

    return params
}

func decodeRPCParams(params string, convertToInt64 bool) (interface{}, error) {
    buffer := bytes.NewBufferString(params)
    decoder := NewJSONDecoder(buffer, convertToInt64)
    var args interface{}
    if err := decoder.Decode(args); err == nil {
        return args, nil
    } else {
        return nil, err
    }
}

func encodeRPCReply(object interface{}, indent bool) (string, error) {
    var buffer bytes.Buffer
    encoder := NewJSONEncoder(&buffer, indent)
    if err := encoder.Encode(object); err == nil {
        return buffer.String(), nil
    } else {
        return "", err
    }
}

func doRPC(host, port, method, params string, options Options) (err error) {
    var (
        args interface{}
        conn net.Conn
    )

    params = adjustRPCParams(params)
    if args, err = decodeRPCParams(params, options.convertToInt64); err != nil {
        return err
    }

    if conn, err = net.Dial("tcp", host+":"+port); err != nil {
        return err
    }
    defer conn.Close()

    result := make(chan RPCResult)
    defer close(result)

    go callRPC(result, conn, method, args)

    select {
    case res := <-result:
        if res.err != nil {
            return res.err
        }

        if data, err := encodeRPCReply(res.reply, options.indent); err == nil {
            fmt.Println(data)
        } else {
            return err
        }
    case <-time.After(time.Duration(options.timeout) * time.Second):
        return fmt.Errorf("RPC call timed out")
    }

    return nil
}

func callRPC(result chan<- RPCResult, conn net.Conn, method string, args interface{}) {
    client := NewMsgpackRPCClient(conn)

    var reply interface{}
    if err := client.Call(method, args, &reply); err != nil {
        result <- RPCResult{reply: nil, err: fmt.Errorf("RPC error: %s", err)}
    }

    result <- RPCResult{reply: reply, err: nil}
}

func convertJSON2Msgpack(reader io.Reader, writer io.Writer, options Options) (err error) {
    var object interface{}

    decoder := NewJSONDecoder(reader, options.convertToInt64)
    encoder := NewMsgpackEncoder(writer)

    for {
        if err = decoder.Decode(&object); err != nil {
            if err == io.EOF {
                break
            } else {
                return err
            }
        }

        if err = encoder.Encode(object); err != nil {
            return err
        }
    }

    return nil
}

func convertMsgpack2JSON(reader io.Reader, writer io.Writer, options Options) (err error) {
    var object interface{}

    decoder := NewMsgpackDecoder(reader)
    encoder := NewJSONEncoder(writer, options.indent)

    for {
        if err = decoder.Decode(&object); err != nil {
            if err == io.EOF {
                break
            } else {
                return err
            }
        }

        if err = encoder.Encode(&object); err != nil {
            return err
        }
    }

    return nil
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
