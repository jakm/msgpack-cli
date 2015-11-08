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
    "io"
    "os"
)

type Encoder interface {
    Encode(v interface{}) error
}

type Decoder interface {
    Decode(v interface{}) error
}

type ConversionFunc func(r io.Reader, w io.Writer, options Options) error

func ConvertFormats(inFilename, outFilename string, conversionFunc ConversionFunc, options Options) error {
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

func ConvertJSON2Msgpack(reader io.Reader, writer io.Writer, options Options) (err error) {
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

func ConvertMsgpack2JSON(reader io.Reader, writer io.Writer, options Options) (err error) {
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

        if err = encoder.Encode(object); err != nil {
            return err
        }
    }

    return nil
}
