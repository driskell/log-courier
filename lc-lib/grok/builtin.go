/*
 * Copyright 2012-2020 Jason Woods and contributors
 *
 * This file contains modified grok patterns, the originals of which are:
 *   Copyright (c) 2012-2018 Elasticsearch <http://www.elastic.co>
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

package grok

var (
	// DefaultPatterns contains a list of default grok patterns built into the binary
	// Most are from logstash-patterns-core as of March 2020
	// Information about any which need adjusting to convert to RE2 compatible versions are noted
	// with the original PCRE versions
	// Most of them are a case of loosening by removing look-behind or look-ahead to prevent incorrect
	// matching positions, but should be fine as the majority of log matchers attempt to match the full line
	// Possessiveness is also removed, and should not be a huge issue as generally tends to be for optimisation
	// of PCRE expressions, something unnecessary with the RE2 FSM
	DefaultPatterns map[string]string = map[string]string{
		"USERNAME":       `[a-zA-Z0-9._-]+`,
		"USER":           `%{USERNAME}`,
		"EMAILLOCALPART": `[a-zA-Z][a-zA-Z0-9_.+-=:]+`,
		"EMAILADDRESS":   `%{EMAILLOCALPART}@%{HOSTNAME}`,
		"INT":            `(?:[+-]?(?:[0-9]+))`,
		// Original PCRE BASE10NUM: (?<![0-9.+-])(?>[+-]?(?:(?:[0-9]+(?:\.[0-9]+)?)|(?:\.[0-9]+)))
		// To convert to RE2, we need to loosen the match and remove the look-behind and the possessive flag
		"BASE10NUM": `(?:[+-]?(?:(?:[0-9]+(?:\.[0-9]+)?)|(?:\.[0-9]+)))`,
		"NUMBER":    `(?:%{BASE10NUM})`,
		// Original PCRE BASE16NUM: (?<![0-9A-Fa-f])(?:[+-]?(?:0x)?(?:[0-9A-Fa-f]+))
		// To convert to RE2, we need to loosen the match and remove the look-behind
		"BASE16NUM": `(?:[+-]?(?:0x)?(?:[0-9A-Fa-f]+))`,
		// Original PCRE BASE16FLOAT: \b(?<![0-9A-Fa-f.])(?:[+-]?(?:0x)?(?:(?:[0-9A-Fa-f]+(?:\.[0-9A-Fa-f]*)?)|(?:\.[0-9A-Fa-f]+)))\b
		// To convert to RE2, remove the look-behind, which loosens only slightly as it's there to prevent matching mantissa only if the non-mantissa was captured in something else
		"BASE16FLOAT": `\b(?:[+-]?(?:0x)?(?:(?:[0-9A-Fa-f]+(?:\.[0-9A-Fa-f]*)?)|(?:\.[0-9A-Fa-f]+)))\b`,

		"POSINT":     `\b(?:[1-9][0-9]*)\b`,
		"NONNEGINT":  `\b(?:[0-9]+)\b`,
		"WORD":       `\b\w+\b`,
		"NOTSPACE":   `\S+`,
		"SPACE":      `\s*`,
		"DATA":       `.*?`,
		"GREEDYDATA": `.*`,
		// Original PCRE QUOTEDSTRING: (?>(?<!\\)(?>"(?>\\.|[^\\"]+)+"|""|(?>'(?>\\.|[^\\']+)+')|''|(?>`(?>\\.|[^\\`]+)+`)|``))`
		// To convert to RE2, remove the look-behind that check we didn't capture an escape in previous match, which might loosen it slightly
		// Also remove the posessiveness
		"QUOTEDSTRING": `(?:(?:"(?:\\.|[^\\"]+)+"|""|(?:'(?:\\.|[^\\']+)+')|''|(?:` + "`" + `(?:\\.|[^\\` + "`]+)+`)|``))`",
		"UUID":         `[A-Fa-f0-9]{8}-(?:[A-Fa-f0-9]{4}-){3}[A-Fa-f0-9]{12}`,
		// URN, allowing use of RFC 2141 section 2.3 reserved characters
		"URN": `urn:[0-9A-Za-z][0-9A-Za-z-]{0,31}:(?:%[0-9a-fA-F]{2}|[0-9A-Za-z()+,.:=@;$_!*'/?#-])+`,

		// Networking
		"MAC":        `(?:%{CISCOMAC}|%{WINDOWSMAC}|%{COMMONMAC})`,
		"CISCOMAC":   `(?:(?:[A-Fa-f0-9]{4}\.){2}[A-Fa-f0-9]{4})`,
		"WINDOWSMAC": `(?:(?:[A-Fa-f0-9]{2}-){5}[A-Fa-f0-9]{2})`,
		"COMMONMAC":  `(?:(?:[A-Fa-f0-9]{2}:){5}[A-Fa-f0-9]{2})`,
		"IPV6":       `((([0-9A-Fa-f]{1,4}:){7}([0-9A-Fa-f]{1,4}|:))|(([0-9A-Fa-f]{1,4}:){6}(:[0-9A-Fa-f]{1,4}|((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){5}(((:[0-9A-Fa-f]{1,4}){1,2})|:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){4}(((:[0-9A-Fa-f]{1,4}){1,3})|((:[0-9A-Fa-f]{1,4})?:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){3}(((:[0-9A-Fa-f]{1,4}){1,4})|((:[0-9A-Fa-f]{1,4}){0,2}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){2}(((:[0-9A-Fa-f]{1,4}){1,5})|((:[0-9A-Fa-f]{1,4}){0,3}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){1}(((:[0-9A-Fa-f]{1,4}){1,6})|((:[0-9A-Fa-f]{1,4}){0,4}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(:(((:[0-9A-Fa-f]{1,4}){1,7})|((:[0-9A-Fa-f]{1,4}){0,5}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:)))(%.+)?`,
		// Original PCRE IPV4: (?<![0-9])(?:(?:[0-1]?[0-9]{1,2}|2[0-4][0-9]|25[0-5])[.](?:[0-1]?[0-9]{1,2}|2[0-4][0-9]|25[0-5])[.](?:[0-1]?[0-9]{1,2}|2[0-4][0-9]|25[0-5])[.](?:[0-1]?[0-9]{1,2}|2[0-4][0-9]|25[0-5]))(?![0-9])
		// To convert to RE2, remove the look-behind and look-ahead which loosens it slightly, it was preventing match after or before another captured number
		"IPV4":     `(?:(?:[0-1]?[0-9]{1,2}|2[0-4][0-9]|25[0-5])[.](?:[0-1]?[0-9]{1,2}|2[0-4][0-9]|25[0-5])[.](?:[0-1]?[0-9]{1,2}|2[0-4][0-9]|25[0-5])[.](?:[0-1]?[0-9]{1,2}|2[0-4][0-9]|25[0-5]))`,
		"IP":       `(?:%{IPV6}|%{IPV4})`,
		"HOSTNAME": `\b(?:[0-9A-Za-z][0-9A-Za-z-]{0,62})(?:\.(?:[0-9A-Za-z][0-9A-Za-z-]{0,62}))*(\.?|\b)`,
		"IPORHOST": `(?:%{IP}|%{HOSTNAME})`,
		"HOSTPORT": `%{IPORHOST}:%{POSINT}`,

		// paths
		"PATH":     `(?:%{UNIXPATH}|%{WINPATH})`,
		"UNIXPATH": `(/([\w_%!$@:.,+~-]+|\\.)*)+`,
		"TTY":      `(?:/dev/(pts|tty([pq])?)(\w+)?/?(?:[0-9]+))`,
		// Original PCRE WINPATH: (?>[A-Za-z]+:|\\)(?:\\[^\\?*]*)+
		// To convert to RE2, remove the possessiveness
		"WINPATH":  `(?:[A-Za-z]+:|\\)(?:\\[^\\?*]*)+`,
		"URIPROTO": `[A-Za-z]([A-Za-z0-9+\-.]+)+`,
		"URIHOST":  `%{IPORHOST}(?::%{POSINT:port})?`,
		// uripath comes loosely from RFC1738, but mostly from what Firefox
		// doesn't turn into %XX
		"URIPATH": `(?:/[A-Za-z0-9$.+!*'(){},~:;=@#%&_\-]*)+`,
		// This is a commented out URIPARAM alternative, possibly it is a stricter alternative that was deemed too strict
		// URIPARAM \?(?:[A-Za-z0-9]+(?:=(?:[^&]*))?(?:&(?:[A-Za-z0-9]+(?:=(?:[^&]*))?)?)*)?
		"URIPARAM":     `\?[A-Za-z0-9$.+!*'|(){},~@#%&/=:;_?\-\[\]<>]*`,
		"URIPATHPARAM": `%{URIPATH}(?:%{URIPARAM})?`,
		"URI":          `%{URIPROTO}://(?:%{USER}(?::[^@]*)?@)?(?:%{URIHOST})?(?:%{URIPATHPARAM})?`,

		// Months: January, Feb, 3, 03, 12, December
		"MONTH":     `\b(?:[Jj]an(?:uary|uar)?|[Ff]eb(?:ruary|ruar)?|[Mm](?:a|ä)?r(?:ch|z)?|[Aa]pr(?:il)?|[Mm]a(?:y|i)?|[Jj]un(?:e|i)?|[Jj]ul(?:y|i)?|[Aa]ug(?:ust)?|[Ss]ep(?:tember)?|[Oo](?:c|k)?t(?:ober)?|[Nn]ov(?:ember)?|[Dd]e(?:c|z)(?:ember)?)\b`,
		"MONTHNUM":  `(?:0?[1-9]|1[0-2])`,
		"MONTHNUM2": `(?:0[1-9]|1[0-2])`,
		"MONTHDAY":  `(?:(?:0[1-9])|(?:[12][0-9])|(?:3[01])|[1-9])`,

		// Days: Monday, Tue, Thu, etc...
		"DAY": `(?:Mon(?:day)?|Tue(?:sday)?|Wed(?:nesday)?|Thu(?:rsday)?|Fri(?:day)?|Sat(?:urday)?|Sun(?:day)?)`,

		// Years?
		// Original PCRE YEAR: (?>\d\d){1,2}
		// To convert to RE2, remove the possessiveness
		"YEAR":   `(?:\d\d){1,2}`,
		"HOUR":   `(?:2[0123]|[01]?[0-9])`,
		"MINUTE": `(?:[0-5][0-9])`,
		// '60' is a leap second in most time standards and thus is valid.
		"SECOND": `(?:(?:[0-5]?[0-9]|60)(?:[:.,][0-9]+)?)`,
		// Original PCRE TIME: (?!<[0-9])%{HOUR}:%{MINUTE}(?::%{SECOND})(?![0-9])
		// To convert to RE2, remove the look-behind and look-ahead that tried to stop it matching after a number caught elsewhere
		// The look-ahead also attempts to prevent it matching incorrectly a MIN:SEC:MSEC I would guess, so does loosen this slightly
		"TIME": `%{HOUR}:%{MINUTE}(?::%{SECOND})`,
		// datestamp is YYYY/MM/DD-HH:MM:SS.UUUU (or something like it)
		"DATE_US":            `%{MONTHNUM}[/-]%{MONTHDAY}[/-]%{YEAR}`,
		"DATE_EU":            `%{MONTHDAY}[./-]%{MONTHNUM}[./-]%{YEAR}`,
		"ISO8601_TIMEZONE":   `(?:Z|[+-]%{HOUR}(?::?%{MINUTE}))`,
		"ISO8601_SECOND":     `(?:%{SECOND}|60)`,
		"TIMESTAMP_ISO8601":  `%{YEAR}-%{MONTHNUM}-%{MONTHDAY}[T ]%{HOUR}:?%{MINUTE}(?::?%{SECOND})?%{ISO8601_TIMEZONE}?`,
		"DATE":               `%{DATE_US}|%{DATE_EU}`,
		"DATESTAMP":          `%{DATE}[- ]%{TIME}`,
		"TZ":                 `(?:[APMCE][SD]T|UTC)`,
		"DATESTAMP_RFC822":   `%{DAY} %{MONTH} %{MONTHDAY} %{YEAR} %{TIME} %{TZ}`,
		"DATESTAMP_RFC2822":  `%{DAY}, %{MONTHDAY} %{MONTH} %{YEAR} %{TIME} %{ISO8601_TIMEZONE}`,
		"DATESTAMP_OTHER":    `%{DAY} %{MONTH} %{MONTHDAY} %{TIME} %{TZ} %{YEAR}`,
		"DATESTAMP_EVENTLOG": `%{YEAR}%{MONTHNUM2}%{MONTHDAY}%{HOUR}%{MINUTE}%{SECOND}`,

		// Syslog Dates: Month Day HH:MM:SS
		"SYSLOGTIMESTAMP": `%{MONTH} +%{MONTHDAY} %{TIME}`,
		"PROG":            `[\x21-\x5a\x5c\x5e-\x7e]+`,
		"SYSLOGPROG":      `%{PROG:program}(?:\[%{POSINT:pid}\])?`,
		"SYSLOGHOST":      `%{IPORHOST}`,
		"SYSLOGFACILITY":  `<%{NONNEGINT:facility}.%{NONNEGINT:priority}>`,
		"HTTPDATE":        `%{MONTHDAY}/%{MONTH}/%{YEAR}:%{TIME} %{INT}`,

		// Shortcuts
		"QS": `%{QUOTEDSTRING}`,

		// Log formats
		"SYSLOGBASE": `%{SYSLOGTIMESTAMP:timestamp} (?:%{SYSLOGFACILITY} )?%{SYSLOGHOST:logsource} %{SYSLOGPROG}:`,

		// Log Levels
		"LOGLEVEL": `([Aa]lert|ALERT|[Tt]race|TRACE|[Dd]ebug|DEBUG|[Nn]otice|NOTICE|[Ii]nfo|INFO|[Ww]arn?(?:ing)?|WARN?(?:ING)?|[Ee]rr?(?:or)?|ERR?(?:OR)?|[Cc]rit?(?:ical)?|CRIT?(?:ICAL)?|[Ff]atal|FATAL|[Ss]evere|SEVERE|EMERG(?:ENCY)?|[Ee]merg(?:ency)?)`,
	}
)
