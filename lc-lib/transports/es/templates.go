/*
 * Copyright 2012-2020 Jason Woods and contributors
 *
 * This file contains modified templates the originals of which are:
 *   Copyright 2012-2018 Elasticsearch <http://www.elastic.co>
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
	// Elasticsearch 8.x
	esTemplate8 = `{
	"index_patterns": $INDEXPATTERNS$,
	"version": 80002,
	"settings": {
		"index.refresh_interval": "5s",
		"number_of_shards": 1
	},
	"mappings": {
		"dynamic_templates": [ {
			"message_field": {
				"path_match": "message",
				"match_mapping_type": "string",
				"mapping": {
					"type": "text",
					"norms": false
				}
			}
		}, {
			"string_fields": {
				"match": "*",
				"match_mapping_type": "string",
				"mapping": {
					"type": "text",
					"norms": false,
					"fields": {
						"keyword": { "type": "keyword", "ignore_above": 8191 }
					}
				}
			}
		} ],
		"properties": {
			"@timestamp": { "type": "date" },
			"event": {
				"dynamic": true,
				"properties": {
					"dataset": { "type": "keyword", "ignore_above": 1024 }
				}
			},
			"host": {
				"dynamic": true,
				"properties": {
					"name": { "type": "keyword", "ignore_above": 1024 },
					"hostname": { "type": "keyword", "ignore_above": 1024 }
				}
			},
			"source": {
				"dynamic": true,
				"properties": {
					"geo": {
						"dynamic": true,
						"properties": {
							"ip": { "type": "ip" },
							"location": { "type": "geo_point" },
							"latitude": { "type": "half_float" },
							"longitude": { "type": "half_float" }
						}
					}
				}
			}
		}
	}
}
`

	// Elasticsearch 7.x (same as 8.x)
	esTemplate7 = esTemplate8

	// Elasticsearch 6.x
	esTemplate6 = `{
	"index_patterns": $INDEXPATTERNS$,
	"version": 60002,
	"settings": {
		"index.refresh_interval": "5s"
	},
	"mappings": {
		"_default_": {
			"dynamic_templates": [ {
				"message_field": {
					"path_match": "message",
					"match_mapping_type": "string",
					"mapping": {
						"type": "text",
						"norms": false
					}
					}
			}, {
				"string_fields": {
					"match": "*",
					"match_mapping_type": "string",
					"mapping": {
						"type": "text",
						"norms": false,
						"fields": {
							"keyword": { "type": "keyword", "ignore_above": 8191 }
						}
					}
				}
			} ],
			"properties": {
				"@timestamp": { "type": "date" },
				"event": {
					"dynamic": true,
					"properties": {
						"dataset": { "type": "keyword", "ignore_above": 1024 }
					}
				},
				"host": {
					"dynamic": true,
					"properties": {
						"name": { "type": "keyword", "ignore_above": 1024 },
						"hostname": { "type": "keyword", "ignore_above": 1024 }
					}
				},
				"source": {
					"dynamic": true,
					"properties": {
						"geo": {
							"dynamic": true,
							"properties": {
								"ip": { "type": "ip" },
								"location": { "type": "geo_point" },
								"latitude": { "type": "half_float" },
								"longitude": { "type": "half_float" }
							}
						}
					}
				}
			}
		}
	}
}`

	// Elasticsearch 5.x
	esTemplate5 = `{
	"template": $INDEXPATTERNSINGLE$,
	"version": 50002,
	"settings": {
		"index.refresh_interval": "5s"
	},
	"mappings": {
		"_default_": {
			"_all": { "enabled": true, "norms": false },
			"dynamic_templates": [ {
				"message_field": {
					"path_match": "message",
					"match_mapping_type": "string",
					"mapping": {
						"type": "text",
						"norms": false
					}
				}
			}, {
				"string_fields": {
					"match": "*",
					"match_mapping_type": "string",
					"mapping": {
						"type": "text",
						"norms": false,
						"fields": {
							"keyword": { "type": "keyword", "ignore_above": 8191 }
						}
					}
				}
			} ],
			"properties": {
				"@timestamp": { "type": "date" },
				"event": {
					"dynamic": true,
					"properties": {
						"dataset": { "type": "keyword", "ignore_above": 1024 }
					}
				},
				"host": {
					"dynamic": true,
					"properties": {
						"name": { "type": "keyword", "ignore_above": 1024 },
						"hostname": { "type": "keyword", "ignore_above": 1024 }
					}
				},
				"source": {
					"dynamic": true,
					"properties": {
						"geo": {
							"dynamic": true,
							"properties": {
								"ip": { "type": "ip" },
								"location": { "type": "geo_point" },
								"latitude": { "type": "half_float" },
								"longitude": { "type": "half_float" }
							}
						}
					}
				}
			}
		}
	}
}`
)
