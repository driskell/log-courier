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

package es

const (
	esTemplate8 = `{
	"index_patterns" : "logstash-*",
	"version" : 80001,
	"settings" : {
		"index.refresh_interval" : "5s",
		"number_of_shards": 1
	},
	"mappings" : {
		"dynamic_templates" : [ {
			"message_field" : {
				"path_match" : "message",
				"match_mapping_type" : "string",
				"mapping" : {
					"type" : "text",
					"norms" : false
				}
			}
		}, {
			"string_fields" : {
				"match" : "*",
				"match_mapping_type" : "string",
				"mapping" : {
					"type" : "text",
					"norms" : false,
					"fields" : {
						"keyword" : { "type": "keyword", "ignore_above": 256 }
					}
				}
			}
		} ],
		"properties" : {
			"@timestamp": { "type": "date"},
			"@version": { "type": "keyword"},
			"geoip"  : {
				"dynamic": true,
				"properties" : {
					"ip": { "type": "ip" },
					"location" : { "type" : "geo_point" },
					"latitude" : { "type" : "half_float" },
					"longitude" : { "type" : "half_float" }
				}
			}
		}
	}
}
`

	esTemplate7 = esTemplate8

	esTemplate6 = `{
	"template" : "logstash-*",
	"version" : 60001,
	"settings" : {
		"index.refresh_interval" : "5s"
	},
	"mappings" : {
		"_default_" : {
			"dynamic_templates" : [ {
				"message_field" : {
					"path_match" : "message",
					"match_mapping_type" : "string",
					"mapping" : {
						"type" : "text",
						"norms" : false
					}
					}
			}, {
				"string_fields" : {
					"match" : "*",
					"match_mapping_type" : "string",
					"mapping" : {
						"type" : "text",
						"norms" : false,
						"fields" : {
							"keyword" : { "type": "keyword", "ignore_above": 256 }
						}
					}
				}
			} ],
			"properties" : {
				"@timestamp": { "type": "date"},
				"@version": { "type": "keyword"},
				"geoip"  : {
					"dynamic": true,
					"properties" : {
						"ip": { "type": "ip" },
						"location" : { "type" : "geo_point" },
						"latitude" : { "type" : "half_float" },
						"longitude" : { "type" : "half_float" }
					}
				}
			}
		}
	}
}`

	esTemplate5 = `{
	"template" : "logstash-*",
	"version" : 50001,
	"settings" : {
		"index.refresh_interval" : "5s"
	},
	"mappings" : {
		"_default_" : {
			"_all" : {"enabled" : true, "norms" : false},
			"dynamic_templates" : [ {
				"message_field" : {
					"path_match" : "message",
					"match_mapping_type" : "string",
					"mapping" : {
						"type" : "text",
						"norms" : false
					}
				}
			}, {
				"string_fields" : {
					"match" : "*",
					"match_mapping_type" : "string",
					"mapping" : {
						"type" : "text",
						"norms" : false,
						"fields" : {
							"keyword" : { "type": "keyword", "ignore_above": 256 }
						}
					}
				}
			} ],
			"properties" : {
				"@timestamp": { "type": "date", "include_in_all": false },
				"@version": { "type": "keyword", "include_in_all": false },
				"geoip"  : {
					"dynamic": true,
					"properties" : {
						"ip": { "type": "ip" },
						"location" : { "type" : "geo_point" },
						"latitude" : { "type" : "half_float" },
						"longitude" : { "type" : "half_float" }
					}
				}
			}
		}
	}
}`
)