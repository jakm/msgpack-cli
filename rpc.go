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
    "bytes"
    "fmt"
    "net"
    "strconv"
    "strings"
    "time"
    "unicode"
    "unicode/utf8"
)

type RPCClient interface {
    Call(serviceMethod string, args interface{}, reply interface{}) error
}

type RPCResult struct {
    reply interface{}
    err   error
}

func CallRPC(host, port, method, params string, options Options) (err error) {
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
    if err := decoder.Decode(&args); err == nil {
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
