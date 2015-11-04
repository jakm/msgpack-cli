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
    "github.com/ugorji/go/codec"
    "io"
    "net"
    "net/rpc"
    "reflect"
    "strings"
)

type Encoder interface {
    Encode(v interface{}) error
}

type Decoder interface {
    Decode(v interface{}) error
}

type RPCClient interface {
    Call(serviceMethod string, args interface{}, reply interface{}) error
}

type indentedJSONEncoder struct {
    w io.Writer
}

func (e *indentedJSONEncoder) Encode(v interface{}) error {
    if data, err := json.MarshalIndent(v, "", "  "); err == nil {
        if _, err := e.w.Write(data); err != nil {
            return err
        }
    } else {
        return err
    }

    return nil
}

type convertingJSONDecoder struct {
    d *json.Decoder
}

type msgpackRPCClient struct {
    c *rpc.Client
}

func (c *msgpackRPCClient) Call(serviceMethod string, args interface{}, reply interface{}) error {
    var mArgs codec.MsgpackSpecRpcMultiArgs = args.([]interface{})
    return c.c.Call(serviceMethod, mArgs, reply)
}

func (d *convertingJSONDecoder) Decode(v interface{}) error {
    if err := d.d.Decode(v); err == nil {
        return convertNumberTypes(&v)
    } else {
        return err
    }
}

func NewMsgpackEncoder(w io.Writer) Encoder {
    h := getHandle()
    return codec.NewEncoder(w, &h)
}

func NewMsgpackDecoder(r io.Reader) Decoder {
    h := getHandle()
    return codec.NewDecoder(r, &h)
}

func NewJSONEncoder(w io.Writer, indent bool) Encoder {
    if indent {
        return &indentedJSONEncoder{w}
    } else {
        return json.NewEncoder(w)
    }
}

func NewJSONDecoder(r io.Reader, convertToInt64 bool) Decoder {
    if convertToInt64 {
        d := json.NewDecoder(r)
        d.UseNumber()
        return &convertingJSONDecoder{d}
    } else {
        return json.NewDecoder(r)
    }
}

func NewMsgpackRPCClient(c net.Conn) RPCClient {
    h := getHandle()
    rpcCodec := codec.MsgpackSpecRpc.ClientCodec(c, &h)
    return rpc.NewClientWithCodec(rpcCodec)
}

func getHandle() codec.MsgpackHandle {
    h := codec.MsgpackHandle{RawToString: true}
    h.MapType = reflect.TypeOf(map[string]interface{}(nil))
    return h
}

func convertNumberTypes(object *interface{}) (err error) {
    switch value := (*object).(type) {
    case json.Number:
        // fmt.Printf("Type %s, value %v\n", reflect.TypeOf(value), value)
        if strings.ContainsAny(value.String(), ".eE") {
            *object, err = value.Float64()
        } else {
            *object, err = value.Int64()
        }
    case []interface{}:
        // fmt.Printf("Type %s, value %v\n", reflect.TypeOf(value), value)
        for idx := range value {
            if err = convertNumberTypes(&value[idx]); err != nil {
                break
            }
        }
    case map[string]interface{}:
        // fmt.Printf("Type %s, value %v\n", reflect.TypeOf(value), value)
        for k, v := range value {
            if err = convertNumberTypes(&v); err != nil {
                break
            } else {
                value[k] = v
            }
        }
    default:
        // fmt.Printf("Type %s, value %v\n", reflect.TypeOf(value), value)
    }

    if err != nil {
        object = nil
    }

    return err
}
