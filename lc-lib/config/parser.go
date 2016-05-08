/*
 * Copyright 2014-2015 Jason Woods.
 *
 * This file is a modification of code from Logstash Forwarder.
 * Copyright 2012-2013 Jordan Sissel and contributors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	"gopkg.in/op/go-logging.v1"
)

// PopulateConfig populates dynamic configuration, automatically converting time.Duration etc.
// Any config entries not found in the structure are moved to an "Unused" field if it exists
// or an error is reported if "Unused" is not available
// We can then take the unused configuration dynamically at runtime based on another value
func PopulateConfig(config interface{}, rawConfig interface{}, configPath string) (err error) {
	// We allow both map[string]interface{} and map[interface{}]interface{}
	// so we will work with reflection values on rawConfig as well as the
	// configuration
	vRawConfig := reflect.ValueOf(rawConfig)
	vConfig := reflect.ValueOf(config)

	return populateStruct(vConfig, vRawConfig, configPath)
}

// PopulateSlice populates dynamic configuration, like PopulateConfig, but by
// appending to a slice instead of writing into a structure
func PopulateSlice(configSlice interface{}, rawConfig []interface{}, configPath string) (err error) {
	vRawConfig := reflect.ValueOf(rawConfig)
	vConfig := reflect.ValueOf(configSlice).Elem()

	return populateSlice(vConfig, vRawConfig, configPath)
}

func populateStruct(vConfig reflect.Value, vRawConfig reflect.Value, configPath string) (err error) {
	// Does the configuration structure have InitDefaults method? Call it to
	// pre-populate the default values before we overwrite the ones given by
	// rawConfig
	if initDefaults := vConfig.MethodByName("InitDefaults"); initDefaults.IsValid() {
		initDefaults.Call([]reflect.Value{})
	}

	// Strip pointers
	vConfig = vConfig.Elem()

	// Iterate each configuration structure field we need to update, and copy the
	// value in, checking the type and removing the value from rawConfig as we use
	// it
FieldLoop:
	for i := 0; i < vConfig.NumField(); i++ {
		vField := vConfig.Field(i)

		// Ensure field is public
		if !vField.CanSet() {
			continue
		}

		// Load the tags from the field's Type
		tField := vConfig.Type().Field(i)
		tag := tField.Tag.Get("config")
		mods := strings.Split(tag, ",")
		tag = mods[0]
		mods = mods[1:]

		// Parse the mods we have on this field
		for _, mod := range mods {
			switch mod {
			case "embed":
				// Embedded field must be public
				if !vField.CanSet() {
					continue
				}

				// Embed means we recurse into the field, but pull it's values from the
				// same level within the configuration file we loaded
				if vField.Kind() != reflect.Struct {
					panic("Embedded configuration field is not a struct")
				}
				if err = populateStruct(vField.Addr(), vRawConfig, configPath); err != nil {
					return
				}
				continue FieldLoop
			case "dynamic":
				// Dynamic means this is a map of Configurables that is dynamically
				// populated with configuration structures at run-time, with the config
				// file key being the map key
				// This is generally not exported so don't check that
				dynamicKeys := vField.MapKeys()
				for _, key := range dynamicKeys {
					// Unwrap the interface and the pointer
					if err = populateEntry(vField.MapIndex(key).Elem().Elem(), vRawConfig, configPath, key.String()); err != nil {
						return
					}
				}
			}
		}

		// If no tag, we're not supposed to read this config entry
		if tag == "" {
			continue
		}

		if err = populateEntry(vField, vRawConfig, configPath, tag); err != nil {
			return
		}
	}

	// Check for unused values in the configuration data and, if there is a field
	// called "Unused" in this structure, store them there. This allows post
	// processing of configuration data for regions of the configuration where
	// the available fields is dynamic (such as within a codec block)
	if unUsed := vConfig.FieldByName("Unused"); unUsed.IsValid() {
		if unUsed.IsNil() {
			unUsed.Set(reflect.MakeMap(unUsed.Type()))
		}
		for _, vKey := range vRawConfig.MapKeys() {
			// If the key is wrapped in interface{}, unwrap it
			if vKey.Type().Kind() == reflect.Interface {
				vKey = vKey.Elem()
			}

			unUsed.SetMapIndex(vKey, vRawConfig.MapIndex(vKey))
		}
		return
	}

	// Report to the user any unused values if there are any, in case they
	// misspelled an option
	return reportUnusedConfig(vRawConfig, configPath)
}

// populateEntry handles population of a single entry, working out whether it
// can assign directly or needs to process as a struct
func populateEntry(vField reflect.Value, vRawConfig reflect.Value, configPath string, tag string) (err error) {
	var vMapIndex reflect.Value

	if vRawConfig.IsValid() {
		// Find the value for this field in the raw configuration data
		vTag := reflect.ValueOf(tag)
		vMapIndex = vRawConfig.MapIndex(vTag)

		// If the map index existed, unwrap the interface{}
		if vMapIndex.IsValid() {
			vMapIndex = vMapIndex.Elem()
		}

		// Remove the used entry
		vRawConfig.SetMapIndex(vTag, reflect.Value{})
	} else {
		// vRawConfig is zero value, so there's no configuration to work with
		// and we're just recursing to set defaults
		vMapIndex = vRawConfig
	}

	if vField.Kind() == reflect.Struct {
		// If we found an entry, check it's the right type, a map
		if vRawConfig.IsValid() && vRawConfig.Type().Kind() != reflect.Map {
			return fmt.Errorf("Option %s%s must be a hash", configPath, tag)
		}

		// Recurse with the new structure and values
		if err := populateStruct(vField.Addr(), vMapIndex, fmt.Sprintf("%s%s/", configPath, tag)); err != nil {
			return err
		}

		return
	}

	// If the configuration data is empty for this section, don't consider any
	// values, leave them as the default
	if !vMapIndex.IsValid() {
		return
	}

	err = populateValue(vField, vMapIndex, configPath, tag)
	return
}

// populateValue handles value to value mappings within a single configuration
// structure, such as maps, slices, and scalar values
func populateValue(vField reflect.Value, vValue reflect.Value, configPath string, tag string) (err error) {
	if vValue.Type().AssignableTo(vField.Type()) {
		vField.Set(vValue)
		return
	}

	if vField.Kind() == reflect.Slice {
		if vValue.Kind() != reflect.Slice {
			err = fmt.Errorf("Option %s%s must be an array", configPath, tag)
			return
		}

		err = populateSlice(vField, vValue, fmt.Sprintf("%s%s/", configPath, tag))
		return
	}

	if vField.Kind() == reflect.Map {
		if vValue.Kind() != reflect.Map {
			err = fmt.Errorf("Option %s%s must be a key-value hash", configPath, tag)
			return
		}

		if vField.IsNil() {
			vField.Set(reflect.MakeMap(vField.Type()))
		}

		for _, vKey := range vValue.MapKeys() {
			// If the key is wrapped in interface{}, unwrap it
			if vKey.Type().Kind() == reflect.Interface {
				vKey = vKey.Elem()
			}

			vItem := vValue.MapIndex(vKey)
			if vItem.Elem().Type().AssignableTo(vField.Type().Elem()) {
				vField.SetMapIndex(vKey, vItem.Elem())
			} else {
				err = fmt.Errorf("Option %s%s must be %s or similar", fmt.Sprintf("%s%s/", configPath, tag), vKey.String(), vField.Type().Elem())
				return
			}
		}
		return
	}

	if vField.Type().String() == "time.Duration" {
		var duration float64
		vDuration := reflect.ValueOf(duration)

		if vValue.Type().AssignableTo(vDuration.Type()) {
			duration = vValue.Float()

			if duration < math.MinInt64 || duration > math.MaxInt64 {
				err = fmt.Errorf("Option %s%s must be a valid numeric or string duration", configPath, tag)
				return
			}

			vField.Set(reflect.ValueOf(time.Duration(int64(duration)) * time.Second))
		} else if vValue.Kind() == reflect.String {
			var parseDuration time.Duration

			if parseDuration, err = time.ParseDuration(vValue.String()); err != nil {
				err = fmt.Errorf("Option %s%s was not understood: %s", configPath, tag, err)
			}

			vField.Set(reflect.ValueOf(parseDuration))
		}

		return
	}

	if vField.Type().String() == "logging.Level" {
		if vValue.Kind() != reflect.String {
			err = fmt.Errorf("Option %s%s is not a valid log level (critical, error, warning, notice, info, debug)", configPath, tag)
			return
		}

		var logLevel logging.Level
		if logLevel, err = logging.LogLevel(vValue.String()); err != nil {
			err = fmt.Errorf("Option %s%s is not a valid log level: %s", configPath, tag, err)
			return
		}

		vField.Set(reflect.ValueOf(logLevel))

		return
	}

	if vField.Kind() == reflect.Int64 || vField.Kind() == reflect.Int {
		var number int

		if vValue.Kind() == reflect.Float64 {
			floatNumber := vValue.Float()
			if math.Floor(floatNumber) != floatNumber {
				err = fmt.Errorf("Option %s%s is not a valid integer (float encountered)", configPath, tag)
				return
			}

			number = int(floatNumber)
		} else if vValue.Kind() == reflect.Int {
			number = int(vValue.Int())
		} else {
			err = fmt.Errorf("Option %s%s is not a valid integer", configPath, tag)
			return
		}

		if vField.Kind() == reflect.Int64 {
			vField.Set(reflect.ValueOf(int64(number)))
		} else {
			vField.Set(reflect.ValueOf(number))
		}

		return
	}

	panic(fmt.Sprintf("Unrecognised configuration structure encountered: %s (Kind: %s)", vField.Type().Name(), vField.Kind().String()))
}

// populateSlice is used to populate an array of configuration structures using
// an array from the configuration file
func populateSlice(vField reflect.Value, vRawConfig reflect.Value, configPath string) (err error) {
	tElem := vField.Type().Elem()
	if tElem.Kind() != reflect.Struct {
		// Simple slice copy
		for i := 0; i < vRawConfig.Len(); i++ {
			// Unwrap interface{} with Elem
			vField.Set(reflect.Append(vField, vRawConfig.Index(i).Elem()))
		}
		return
	}

	for i := 0; i < vRawConfig.Len(); i++ {
		vItem := reflect.New(tElem)

		// Unwrap vItem from its pointer, and unwrap the map from it's interface{}
		if err = populateStruct(vItem, vRawConfig.Index(i).Elem(), fmt.Sprintf("%s[%d]/", configPath, i)); err != nil {
			return
		}

		vField.Set(reflect.Append(vField, vItem.Elem()))
	}

	return
}

// ReportUnusedConfig returns an error if the given configuration map is not
// empty. This is used to report unrecognised configuration entries. As each
// configuration entry is mapped into the configuration it is removed from the
// configuration map, so it is expected to end up empty.
func ReportUnusedConfig(rawConfig map[string]interface{}, configPath string) (err error) {
	return reportUnusedConfig(reflect.ValueOf(rawConfig), configPath)
}

// reportUnusedConfig is the internal representation of ReportUnusedConfig that
// works with reflection
func reportUnusedConfig(vRawConfig reflect.Value, configPath string) (err error) {
	if !vRawConfig.IsValid() {
		// Zero value, which means there's no data
		return nil
	}

	for _, vKey := range vRawConfig.MapKeys() {
		// If the key is wrapped in interface{}, unwrap it
		if vKey.Type().Kind() == reflect.Interface {
			vKey = vKey.Elem()
		}

		err = fmt.Errorf("Option %s%s is not available", configPath, vKey.String())
		return
	}
	return
}
