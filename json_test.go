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
    "encoding/json"
    "reflect"
    "testing"
)

func TestConvertNumberTypes(t *testing.T) {
    testConversionOfScalarValue(t)
    testRecursiveConversionOfSlice(t)
    testRecursiveConversionOfMap(t)
}

func testConversionOfScalarValue(t *testing.T) {
    var (
        err    error
        object interface{}
    )

    for _, num := range []string{"0.0", ".0", "-4.", "3.14159", "-4.8e-8", "34e15"} {
        object = json.Number(num)

        err = convertNumberTypes(&object)

        if err != nil {
            t.Fatalf("Conversion of value \"%s\" failed: %s", num, err)
        }

        if reflect.TypeOf(object).Kind() != reflect.Float64 {
            t.Fatalf("Value \"%s\" was was converted to %s type (float64 expected).",
                num, reflect.TypeOf(object))
        }
    }

    for _, num := range []string{"0", "256", "-3", "123456789", "1234567890000"} {
        object = json.Number(num)

        err = convertNumberTypes(&object)

        if err != nil {
            t.Fatalf("Conversion of value \"%s\" failed: %s", num, err)
        }

        if reflect.TypeOf(object).Kind() != reflect.Int64 {
            t.Fatalf("Value \"%s\" was was converted to %s type (int64 expected).",
                num, reflect.TypeOf(object))
        }
    }
}

func testRecursiveConversionOfSlice(t *testing.T) {
    var object interface{}

    object = []interface{}{
        "string",
        json.Number("1234567890000"),
        []interface{}{
            json.Number("256"),
            json.Number(".0"),
            "string",
        },
    }

    checkType := func(object interface{}, expectedType reflect.Kind) {
        if reflect.TypeOf(object).Kind() != expectedType {
            t.Fatalf("Type %s of value %v doesn't match expected type %s",
                reflect.TypeOf(object), object, expectedType)
        }
    }

    if err := convertNumberTypes(&object); err != nil {
        t.Fatalf("Conversion of slice %v failed: %s", object, err)
    } else {
        switch object := object.(type) {
        case []interface{}:
            checkType(object[0], reflect.String)
            checkType(object[1], reflect.Int64)
            checkType(object[2], reflect.Slice)
            switch inner := object[2].(type) {
            case []interface{}:
                checkType(inner[0], reflect.Int64)
                checkType(inner[1], reflect.Float64)
                checkType(inner[2], reflect.String)
            default:
                t.Fatalf("Conversion function returned incorrect type %s of inner slice (expected: []interface{})",
                    reflect.TypeOf(inner))
            }
        default:
            t.Fatalf("Conversion function returned incorrect type %s (expected: []interface{})",
                reflect.TypeOf(object))
        }
    }
}

func testRecursiveConversionOfMap(t *testing.T) {
    var object interface{}

    object = map[string]interface{}{
        "aaa": "string",
        "bbb": json.Number("1234567890000"),
        "ccc": json.Number("4."),
        "ddd": map[string]interface{}{
            "xxx": json.Number("256"),
            "yyy": json.Number(".0"),
            "zzz": "string",
        },
    }

    checkType := func(object interface{}, expectedType reflect.Kind) {
        if reflect.TypeOf(object).Kind() != expectedType {
            t.Fatalf("Type %s of value %v doesn't match expected type %s",
                reflect.TypeOf(object), object, expectedType)
        }
    }

    if err := convertNumberTypes(&object); err != nil {
        t.Fatalf("Conversion of map %v failed: %s", object, err)
    } else {
        switch object := object.(type) {
        case map[string]interface{}:
            checkType(object["aaa"], reflect.String)
            checkType(object["bbb"], reflect.Int64)
            checkType(object["ccc"], reflect.Float64)
            checkType(object["ddd"], reflect.Map)
            switch inner := object["ddd"].(type) {
            case map[string]interface{}:
                checkType(inner["xxx"], reflect.Int64)
                checkType(inner["yyy"], reflect.Float64)
                checkType(inner["zzz"], reflect.String)
            default:
                t.Fatalf("Conversion function returned incorrect type %s of inner map (expected: map[string]interface{})",
                    reflect.TypeOf(inner))
            }
        default:
            t.Fatalf("Conversion function returned incorrect type %s (expected: map[string]interface{})",
                reflect.TypeOf(object))
        }
    }
}
