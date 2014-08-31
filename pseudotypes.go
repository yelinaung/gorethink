package gorethink

import (
	"encoding/base64"
	"math"
	"strconv"
	"time"

	"fmt"
)

func convertPseudotype(obj map[string]interface{}, opts map[string]interface{}) (interface{}, error) {
	if reqlType, ok := obj["$reql_type$"]; ok {
		if reqlType == "TIME" {
			// load timeFormat, set to native if the option was not set
			timeFormat := "native"
			if opt, ok := opts["time_format"]; ok {
				if sopt, ok := opt.(string); ok {
					timeFormat = sopt
				} else {
					return nil, fmt.Errorf("Invalid time_format run option \"%s\".", opt)
				}
			}

			if timeFormat == "native" {
				return reqlTimeToNativeTime(obj["epoch_time"].(float64), obj["timezone"].(string))
			} else if timeFormat == "raw" {
				return obj, nil
			} else {
				return nil, fmt.Errorf("Unknown time_format run option \"%s\".", reqlType)
			}
		} else if reqlType == "GROUPED_DATA" {
			// load groupFormat, set to native if the option was not set
			groupFormat := "native"
			if opt, ok := opts["group_format"]; ok {
				if sopt, ok := opt.(string); ok {
					groupFormat = sopt
				} else {
					return nil, fmt.Errorf("Invalid group_format run option \"%s\".", opt)
				}
			}

			if groupFormat == "native" {
				return reqlGroupedDataToObj(obj)
			} else if groupFormat == "raw" {
				return obj, nil
			} else {
				return nil, fmt.Errorf("Unknown group_format run option \"%s\".", reqlType)
			}
		} else if reqlType == "BINARY" {
			binaryFormat := "native"
			if opt, ok := opts["binary_format"]; ok {
				if sopt, ok := opt.(string); ok {
					binaryFormat = sopt
				} else {
					return nil, fmt.Errorf("Invalid binary_format run option \"%s\".", opt)
				}
			}

			if binaryFormat == "native" {
				return reqlBinaryToNativeBytes(obj)
			} else if binaryFormat == "raw" {
				return obj, nil
			} else {
				return nil, fmt.Errorf("Unknown binary_format run option \"%s\".", reqlType)
			}
		} else {
			return obj, nil
		}
	}

	return obj, nil
}

func recursivelyConvertPseudotype(obj interface{}, opts map[string]interface{}) (interface{}, error) {
	var err error

	switch obj := obj.(type) {
	case []interface{}:
		for key, val := range obj {
			obj[key], err = recursivelyConvertPseudotype(val, opts)
			if err != nil {
				return nil, err
			}
		}
	case map[string]interface{}:
		for key, val := range obj {
			obj[key], err = recursivelyConvertPseudotype(val, opts)
			if err != nil {
				return nil, err
			}
		}

		pobj, err := convertPseudotype(obj, opts)
		if err != nil {
			return nil, err
		}

		return pobj, nil
	}

	return obj, nil
}

// Pseudo-type helper functions

func reqlTimeToNativeTime(timestamp float64, timezone string) (time.Time, error) {
	sec, ms := math.Modf(timestamp)

	t := time.Unix(int64(sec), int64(ms*1000*1000*1000))

	// Caclulate the timezone
	if timezone != "" {
		hours, err := strconv.Atoi(timezone[1:3])
		if err != nil {
			return time.Time{}, err
		}
		minutes, err := strconv.Atoi(timezone[4:6])
		if err != nil {
			return time.Time{}, err
		}
		tzOffset := ((hours * 60) + minutes) * 60
		if timezone[:1] == "-" {
			tzOffset = 0 - tzOffset
		}

		t = t.In(time.FixedZone(timezone, tzOffset))
	}

	return t, nil
}

func reqlGroupedDataToObj(obj map[string]interface{}) (interface{}, error) {
	if data, ok := obj["data"]; ok {
		ret := []interface{}{}
		for _, v := range data.([]interface{}) {
			v := v.([]interface{})
			ret = append(ret, map[string]interface{}{
				"group":     v[0],
				"reduction": v[1],
			})
		}
		return ret, nil
	} else {
		return nil, fmt.Errorf("pseudo-type GROUPED_DATA object %v does not have the expected field \"data\"", obj)
	}
}

func reqlBinaryToNativeBytes(obj map[string]interface{}) (interface{}, error) {
	if data, ok := obj["data"]; ok {
		if data, ok := data.(string); ok {
			b, err := base64.StdEncoding.DecodeString(data)
			if err != nil {
				return nil, fmt.Errorf("error decoding pseudo-type BINARY object %v", obj)
			}

			return b, nil
		} else {
			return nil, fmt.Errorf("pseudo-type BINARY object %v field \"data\" is not valid", obj)
		}
	} else {
		return nil, fmt.Errorf("pseudo-type BINARY object %v does not have the expected field \"data\"", obj)
	}
}