/*
 * Copyright 2012-2020 Jason Woods and contributors
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
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	"gopkg.in/op/go-logging.v1"
)

// Parser holds the parsing state for configuration population
type Parser struct {
	cfg             *Config
	validationFuncs []reflect.Value
	validationPaths []string
}

// NewParser returns a new parser for the given configuration structure
func NewParser(cfg *Config) *Parser {
	return &Parser{cfg: cfg}
}

// parseConfiguration is a bootstrap around Parser
func parseConfiguration(cfg *Config, rawConfig interface{}, reportUnused bool) error {
	p := NewParser(cfg)
	if err := p.Populate(cfg, rawConfig, "/", reportUnused); err != nil {
		return err
	}

	err := p.validate()
	if log.IsEnabledFor(logging.DEBUG) {
		json, err := json.MarshalIndent(cfg, "", "\t")
		if err == nil {
			log.Debugf("Final configuration: %s", json)
		} else {
			log.Debugf("Final configuration could not be rendered as JSON: %s", err)
		}
	}
	return err
}

// Config returns the root Config currently being parsed
func (p *Parser) Config() *Config {
	return p.cfg
}

// Populate populates dynamic configuration, automatically converting time.Duration etc.
// Any config entries not found in the structure are moved to an "Unused" field if it exists
// or an error is reported if "Unused" is not available
// We can then take the unused configuration dynamically at runtime based on another value
func (p *Parser) Populate(config interface{}, rawConfig interface{}, configPath string, reportUnused bool) (err error) {
	// We allow both map[string]interface{} and map[interface{}]interface{}
	// so we will work with reflection values on rawConfig as well as the
	// configuration
	vRawConfig := reflect.ValueOf(rawConfig)
	vConfig := reflect.ValueOf(config)

	return p.populateStruct(vConfig, vRawConfig, configPath, reportUnused)
}

// PopulateSlice populates dynamic configuration, like Populate, but by
// appending to a slice instead of writing into a structure
func (p *Parser) PopulateSlice(configSlice interface{}, rawConfig []interface{}, configPath string) (err error) {
	vRawConfig := reflect.ValueOf(rawConfig)
	vConfig := reflect.ValueOf(configSlice).Elem()

	var retSlice reflect.Value
	retSlice, err = p.populateSlice(vConfig, vRawConfig, configPath)
	if err == nil {
		vConfig.Set(retSlice)
	}
	return
}

// validate calls all the structure validations
func (p *Parser) validate() (err error) {
	validationFuncs, validationPaths := p.validationFuncs, p.validationPaths
	p.validationFuncs, p.validationPaths = nil, nil

	for k, validateFunc := range validationFuncs {
		log.Debugf("Calling validation: %s", validationPaths[k])
		result := validateFunc.Call([]reflect.Value{
			reflect.ValueOf(p),
			reflect.ValueOf(validationPaths[k]),
		})
		resultErr := result[0].Interface()
		if resultErr != nil {
			return resultErr.(error)
		}
	}

	// Recurse if the validations triggered extra configuration parsing
	if p.validationFuncs != nil {
		return p.validate()
	}

	return
}

// populateStruct populates a structure, and calls its lifecycle events
func (p *Parser) populateStruct(vConfig reflect.Value, vRawConfig reflect.Value, configPath string, reportUnused bool) (err error) {
	log.Debugf("populateStruct: %s (%s)", vConfig.Type().String(), configPath)

	// Initialise defaults and register any validation function
	p.prepareValue(vConfig, configPath)

	if err = p.populateStructInner(vConfig, vRawConfig, configPath); err != nil {
		return
	}

	// Call the Init if any, this should take away the unused values by populating
	// further structures depending on other values
	if err = p.callInit(vConfig, configPath); err != nil {
		return
	}

	// Report to the user any unused values if there are any, in case they
	// misspelled an option
	if reportUnused {
		return p.reportUnusedConfig(vRawConfig, configPath)
	}

	return
}

// populateStructInner handles structure population and also creation in case of pointers
func (p *Parser) populateStructInner(vConfig reflect.Value, vRawConfig reflect.Value, configPath string) (err error) {
	if vConfig.Kind() == reflect.Ptr {
		return p.populateStructInner(vConfig.Elem(), vRawConfig, configPath)
	}

	if vConfig.Kind() != reflect.Struct {
		panic(fmt.Sprintf("Object passed to populateStruct is not a struct: %s", vConfig.Kind().String()))
	}

	// Check the incoming data is the right type, a map
	if vRawConfig.IsValid() && vRawConfig.Type().Kind() != reflect.Map {
		return fmt.Errorf("Option %s must be a hash", configPath)
	}

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
				// Embed means we recurse into the field, but pull it's values from the
				// same level within the configuration file we loaded
				if vField.Kind() != reflect.Struct {
					panic(fmt.Sprintf("Embedded configuration field is not a struct: %s", vField.Kind().String()))
				}

				// Call with pointer to enable lifecycle methods
				if err = p.populateStruct(vField.Addr(), vRawConfig, configPath, false); err != nil {
					return
				}
				continue FieldLoop
			case "dynamic":
				// Dynamic means this is a map of Configurables that is dynamically
				// populated with configuration structures at run-time, with the config
				// file key being the map key
				// This is generally not exported so don't check that
				if vField.Kind() != reflect.Map {
					panic(fmt.Sprintf("Dynamic configuration field is not a map: %s", vField.Kind().String()))
				}

				dynamicKeys := vField.MapKeys()
				for _, key := range dynamicKeys {
					var retValue reflect.Value
					if retValue, err = p.populateEntry(vField.MapIndex(key).Elem(), vRawConfig, configPath, key.String()); err != nil {
						return
					}
					// If nothing was provided, don't store anything, so we keep defaults
					if retValue.IsValid() {
						vField.SetMapIndex(key, retValue)
					}
				}
				continue FieldLoop
			case "embed_dynamic":
				// Embed dynamic is the same as dynamic, except we ignore the keys and
				// dynamically populate each entry as if it were embedded. Used by
				// General to allow packages to add extra general configuration entries
				// without needing to create new configuration sections
				// This means all values of the map should be structs
				if vField.Kind() != reflect.Map {
					panic(fmt.Sprintf("Embedded dynamic configuration field is not a map: %s", vField.Kind().String()))
				}

				dynamicKeys := vField.MapKeys()
				for _, key := range dynamicKeys {
					if err = p.populateStruct(vField.MapIndex(key).Elem(), vRawConfig, configPath, false); err != nil {
						return
					}
				}
				continue FieldLoop
			}
		}

		// If no tag, we're not supposed to read this config entry
		if tag == "" {
			continue
		}

		var retValue reflect.Value
		if retValue, err = p.populateEntry(vField, vRawConfig, configPath, tag); err != nil {
			return
		}

		// If we didn't have it in the provided configuration, zero value is returned
		if retValue.IsValid() {
			vField.Set(retValue)
		}
	}

	// Check for unused values in the configuration data and, if there is a field
	// called "Unused" in this structure, store them there. This allows post
	// processing of configuration data for regions of the configuration where
	// the available fields is dynamic (such as within a codec block)
	if unUsed := vConfig.FieldByName("Unused"); unUsed.IsValid() {
		log.Debugf("Saving unused configuration entries: %s", configPath)
		if unUsed.IsNil() {
			unUsed.Set(reflect.MakeMap(unUsed.Type()))
		}
		if vRawConfig.IsValid() {
			for _, vKey := range vRawConfig.MapKeys() {
				// If the key is wrapped in interface{}, unwrap it
				if vKey.Type().Kind() == reflect.Interface {
					vKey = vKey.Elem()
				}

				unUsed.SetMapIndex(vKey, vRawConfig.MapIndex(vKey))
				vRawConfig.SetMapIndex(vKey, reflect.Value{})
			}
		}
	}

	return
}

// getMapIndex returns the reflect value for the given entry in the incoming configuration
func (p *Parser) getMapIndex(vRawConfig reflect.Value, tag string) reflect.Value {
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

	return vMapIndex
}

// populateEntry handles population of a single entry, working out whether it
// can assign directly or needs to process as a struct
func (p *Parser) populateEntry(vField reflect.Value, vRawConfig reflect.Value, configPath string, tag string) (retValue reflect.Value, err error) {
	log.Debugf("populateEntry: %s (%s%s)", vField.Type().String(), configPath, tag)

	// Strip pointers - but only if not a ptr to struct
	// That way populateStruct will receive the ptr to struct and can call lifecylce events attaches to the ptr version
	if vField.Kind() == reflect.Ptr && vField.Elem().Kind() != reflect.Struct {
		if !vField.Elem().IsValid() {
			vField = reflect.New(vField.Type().Elem())
		}
		var innerValue reflect.Value
		if innerValue, err = p.populateEntry(vField.Elem(), vRawConfig, configPath, tag); err != nil {
			return
		}
		retValue = innerValue.Addr()
		return
	}

	vMapIndex := vRawConfig
	if tag != "" {
		vMapIndex = p.getMapIndex(vRawConfig, tag)
	}

	if vField.Kind() == reflect.Struct || (vField.Kind() == reflect.Ptr && vField.Elem().Kind() == reflect.Struct) {
		retValue = vField
		ptrValue := retValue
		if vField.Kind() == reflect.Ptr {
			if !vField.Elem().IsValid() {
				retValue = reflect.New(vField.Type().Elem())
			}
		} else {
			ptrValue = ptrValue.Addr()
		}
		// Call with pointer to enable lifecycle methods if any
		if err := p.populateStruct(ptrValue, vMapIndex, fmt.Sprintf("%s%s/", configPath, tag), true); err != nil {
			return reflect.Value{}, err
		}
		return
	}

	if vField.Kind() == reflect.Slice {
		retValue, err = p.populateSlice(vField, vMapIndex, fmt.Sprintf("%s%s", configPath, tag))
		return
	}

	// If the configuration data is empty for this section, don't consider any
	// values, leave them as the default
	// Do not skip slice or struct (see above) as they could have lifecycle methods attached
	if !vMapIndex.IsValid() {
		return
	}

	if vMapIndex.Type().AssignableTo(vField.Type()) {
		log.Debugf("populateEntry value: %v (%s%s)", vMapIndex.String(), configPath, tag)
		retValue = vMapIndex
		return
	}

	if vField.Kind() == reflect.Map {
		if vMapIndex.Kind() != reflect.Map {
			err = fmt.Errorf("Option %s%s must be a key-value hash", configPath, tag)
			return
		}

		if vField.IsNil() {
			retValue = reflect.MakeMap(vField.Type())
		}

		for _, vKey := range vMapIndex.MapKeys() {
			// If the key is wrapped in interface{}, unwrap it
			if vKey.Type().Kind() == reflect.Interface {
				vKey = vKey.Elem()
			}

			vItem := vMapIndex.MapIndex(vKey)
			if vItem.Elem().Type().AssignableTo(vField.Type().Elem()) {
				log.Debugf("populateEntry value: map[%s][%s] (%s%s)", vKey.String(), vItem.Elem().String(), configPath, tag)
				retValue.SetMapIndex(vKey, vItem.Elem())
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

		if vMapIndex.Type().AssignableTo(vDuration.Type()) {
			duration = vMapIndex.Float()

			if duration < math.MinInt64 || duration > math.MaxInt64 {
				err = fmt.Errorf("Option %s%s must be a valid numeric or string duration", configPath, tag)
				return
			}

			log.Debugf("populateEntry value: %f (%s%s)", duration, configPath, tag)
			retValue = reflect.ValueOf(time.Duration(int64(duration)) * time.Second)
		} else if vMapIndex.Kind() == reflect.String {
			var parseDuration time.Duration

			if parseDuration, err = time.ParseDuration(vMapIndex.String()); err != nil {
				err = fmt.Errorf("Option %s%s was not understood: %s", configPath, tag, err)
			}

			log.Debugf("populateEntry value: %v (%s%s)", parseDuration, configPath, tag)
			retValue = reflect.ValueOf(parseDuration)
		} else {
			err = fmt.Errorf("Option %s%s is not a valid duration (number of seconds or duration syntax)", configPath, tag)
			return
		}

		return
	}

	if vField.Type().String() == "logging.Level" {
		if vMapIndex.Kind() != reflect.String {
			err = fmt.Errorf("Option %s%s is not a valid log level (critical, error, warning, notice, info, debug)", configPath, tag)
			return
		}

		var logLevel logging.Level
		if logLevel, err = logging.LogLevel(vMapIndex.String()); err != nil {
			err = fmt.Errorf("Option %s%s is not a valid log level: %s", configPath, tag, err)
			return
		}

		log.Debugf("populateEntry value: %v (%s%s)", logLevel, configPath, tag)
		retValue = reflect.ValueOf(logLevel)

		return
	}

	if vField.Kind() == reflect.Int64 || vField.Kind() == reflect.Int {
		var number int

		if vMapIndex.Kind() == reflect.Float64 {
			floatNumber := vMapIndex.Float()
			if math.Floor(floatNumber) != floatNumber {
				err = fmt.Errorf("Option %s%s is not a valid integer (float encountered)", configPath, tag)
				return
			}

			number = int(floatNumber)
		} else if vMapIndex.Kind() == reflect.Int {
			number = int(vMapIndex.Int())
		} else {
			err = fmt.Errorf("Option %s%s is not a valid integer", configPath, tag)
			return
		}

		log.Debugf("populateEntry value: %d (%s%s)", number, configPath, tag)
		if vField.Kind() == reflect.Int64 {
			retValue = reflect.ValueOf(int64(number))
		} else {
			retValue = reflect.ValueOf(number)
		}

		return
	}

	panic(fmt.Sprintf("Unrecognised configuration structure encountered: %s (Kind: %s)", vField.Type().Name(), vField.Kind().String()))
}

// populateSlice is used to populate an array of configuration structures using
// an array from the configuration file
func (p *Parser) populateSlice(vSlice reflect.Value, vRawConfig reflect.Value, configPath string) (retSlice reflect.Value, err error) {
	log.Debugf("populateSlice: %s (%s)", vSlice.Type().String(), configPath)

	if vRawConfig.IsValid() && vRawConfig.Kind() != reflect.Slice {
		err = fmt.Errorf("Option %s must be an array", configPath)
		return
	}

	// Setup default value and register any validation
	p.prepareValue(vSlice, configPath)

	if vSlice.IsZero() {
		vSlice = reflect.MakeSlice(vSlice.Type(), 0, 0)
	}

	if vRawConfig.IsValid() {
		for i := 0; i < vRawConfig.Len(); i++ {
			vItem := reflect.New(vSlice.Type().Elem()).Elem()
			// Dereference interface{} map value in incoming config to get the real item
			configItem := vRawConfig.Index(i)
			if configItem.Kind() == reflect.Interface {
				configItem = configItem.Elem()
			}
			var retValue reflect.Value
			if retValue, err = p.populateEntry(vItem, configItem, fmt.Sprintf("%s[%d]", configPath, i), ""); err != nil {
				return
			}
			vSlice = reflect.Append(vSlice, retValue)
		}
	}

	// Call the Init if any, this should take away the unused values by populating
	// further structures depending on other values
	if err = p.callInit(vSlice, configPath); err != nil {
		return
	}

	retSlice = vSlice
	return
}

// prepareValue calls the defaults function, if any, and also registers any
// validation functions
func (p *Parser) prepareValue(value reflect.Value, configPath string) {
	// Does the configuration structure have InitDefaults method? Call it to
	// pre-populate the default values before we overwrite the ones given by
	// rawConfig
	if defaultsFunc := value.MethodByName("Defaults"); defaultsFunc.IsValid() {
		log.Debugf("Initialising defaults: %s (%s)", value.Type().String(), configPath)
		defaultsFunc.Call([]reflect.Value{})
	}

	// Queue the structure for a Validate call at the end if it has one
	if validateFunc := value.MethodByName("Validate"); validateFunc.IsValid() {
		log.Debugf("Registering validation: %s (%s)", value.Type().String(), configPath)
		p.validationFuncs = append(p.validationFuncs, validateFunc)
		p.validationPaths = append(p.validationPaths, configPath)
	}
}

// callInit calls the custom initialisation function, if any, for the given
// value
func (p *Parser) callInit(value reflect.Value, configPath string) error {
	initFunc := value.MethodByName("Init")
	if !initFunc.IsValid() {
		return nil
	}

	log.Debugf("Calling initialisation: %s (%s)", value.Type().String(), configPath)
	result := initFunc.Call([]reflect.Value{
		reflect.ValueOf(p),
		reflect.ValueOf(configPath),
	})
	resultErr := result[0].Interface()
	if resultErr != nil {
		return resultErr.(error)
	}

	return nil
}

// ReportUnusedConfig returns an error if the given configuration map is not
// empty. This is used to report unrecognised configuration entries. As each
// configuration entry is mapped into the configuration it is removed from the
// configuration map, so it is expected to end up empty.
func (p *Parser) ReportUnusedConfig(rawConfig map[string]interface{}, configPath string) (err error) {
	return p.reportUnusedConfig(reflect.ValueOf(rawConfig), configPath)
}

// reportUnusedConfig is the internal representation of ReportUnusedConfig that
// works with reflection
func (p *Parser) reportUnusedConfig(vRawConfig reflect.Value, configPath string) (err error) {
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

// FixMapKeys converts any map entries where the keys are interface{} values
// into map entries where the key is a string. It returns an error if any key is
// found that is not a string.
// This is important as json.Encode will not encode a map where the keys are not
// concrete strings.
func (p *Parser) FixMapKeys(path string, value map[string]interface{}) error {
	for k, v := range value {
		switch vt := v.(type) {
		case map[string]interface{}:
			if err := p.FixMapKeys(path+"/"+k, vt); err != nil {
				return err
			}
		case map[interface{}]interface{}:
			fixedValue, err := p.fixMapInterfaceKeys(path+"/"+k, vt)
			if err != nil {
				return err
			}

			value[k] = fixedValue
		}
	}

	return nil
}

func (p *Parser) fixMapInterfaceKeys(path string, value map[interface{}]interface{}) (map[string]interface{}, error) {
	fixedMap := make(map[string]interface{})

	for k, v := range value {
		ks, ok := k.(string)
		if !ok {
			return nil, fmt.Errorf("Invalid non-string key at %s", path)
		}

		switch vt := v.(type) {
		case map[string]interface{}:
			if err := p.FixMapKeys(path+"/"+ks, vt); err != nil {
				return nil, err
			}

			fixedMap[ks] = vt
		case map[interface{}]interface{}:
			fixedValue, err := p.fixMapInterfaceKeys(path+"/"+ks, vt)
			if err != nil {
				return nil, err
			}

			fixedMap[ks] = fixedValue
		default:
			fixedMap[ks] = vt
		}
	}

	return fixedMap, nil
}
