# GeoIP Action

The `geoip` action parses an IP address into coordinates using a MaxMind GeoIP database.

- [GeoIP Action](#geoip-action)
  - [Example](#example)
  - [Options](#options)
    - [`database`](#database)
    - [`field`](#field)
    - [`target`](#target)

## Example

```yaml
- name: geoip
  field: address
```

## Options

### `database`

Filepath. Optional. Default system dependant

The path to the MaxMind GeoIP City database. The packaged versions of Log Carver depend on a package that will install this database for you, and the configuration will default to using that database. You can override the path using this configuration. Where Log Carver is built manually this becomes required.

### `field`

String. Required

The name of the field to parse. Use `[]` to access nested fields, for example `nested[field]`.

### `target`

String. Optional. Default: "source[geo]"

Sets the target field to store the GeoIP results into. The default ensures that the `location` GeoJSON coordinates follow the Elastic Common Schema (ECS), but the remaining fields are non-standard. The target field will contain the following nested fields after a successful parse.

- `city_name`. String
- `continent_name`. String
- `country_iso_code`. String
- `country_name`. String
- `location`. GeoJSON (array of two float coordinates, longitude and latitude respectively)
- `latitude`. Float
- `longitude`. Float
- `postal_code`. String
- `timezone`. String
- `region_iso_code`. String. Omitted if no data available
- `region_name`. String. Omitted if no data available

All fields are present with the exception of the region fields which are only present if data is available.
