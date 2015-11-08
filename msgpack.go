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
    "github.com/ugorji/go/codec"
    "io"
    "net"
    "net/rpc"
    "reflect"
)

type msgpackRPCClient struct {
    c *rpc.Client
}

func (c *msgpackRPCClient) Call(serviceMethod string, args interface{}, reply interface{}) error {
    var mArgs codec.MsgpackSpecRpcMultiArgs = args.([]interface{})
    return c.c.Call(serviceMethod, mArgs, reply)
}

func NewMsgpackEncoder(w io.Writer) Encoder {
    h := getHandle()
    return codec.NewEncoder(w, &h)
}

func NewMsgpackDecoder(r io.Reader) Decoder {
    h := getHandle()
    return codec.NewDecoder(r, &h)
}

func NewMsgpackRPCClient(c net.Conn) RPCClient {
    h := getHandle()
    rpcCodec := codec.MsgpackSpecRpc.ClientCodec(c, &h)
    rc := rpc.NewClientWithCodec(rpcCodec)
    return &msgpackRPCClient{rc}
}

func getHandle() codec.MsgpackHandle {
    h := codec.MsgpackHandle{RawToString: true}
    h.MapType = reflect.TypeOf(map[string]interface{}(nil))
    return h
}
