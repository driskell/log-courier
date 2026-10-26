#!/bin/bash

# Runs the pipeline and related micro benchmarks and compares them with the committed reference report using benchstat
#
# Usage: run.sh [--diff | --update-reference]
#   (no option)          run the benchmarks into latest.txt, then compare it with reference.txt
#   --diff               compare latest.txt with reference.txt without running
#   --update-reference   replace reference.txt with latest.txt
#
# COUNT sets the runs per benchmark (default 10) and THRESHOLD the percentage change highlighted (default 10)

set -eo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$DIR/../.." && pwd)"
COUNT="${COUNT:-10}"
THRESHOLD="${THRESHOLD:-10}"
BENCHSTAT="golang.org/x/perf/cmd/benchstat@v0.0.0-20261009192801-be2c69fb417e"

usage() {
	sed -n '5,8p' "$0" | sed 's/^# //'
}

run() {
	cd "$ROOT"

	local commit
	commit="$(git rev-parse HEAD)"
	if ! git diff --quiet HEAD; then
		commit="$commit-modified"
	fi

	{
		echo "commit: $commit"
		echo "date: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
		echo "go: $(go env GOVERSION)"
		echo
		go test ./lc-lib/event ./lc-lib/spooler ./lc-lib/transports/tcp/courier -run '^$' -bench . -benchmem -count "$COUNT"
		go test ./lc-lib/pipelinebench -run '^$' -bench Throughput -benchmem -count "$COUNT"
		go test ./lc-lib/pipelinebench -run '^$' -bench Fairness -benchtime 1x -count "$COUNT"
	} | tee "$DIR/latest.txt.tmp"

	mv "$DIR/latest.txt.tmp" "$DIR/latest.txt"
	echo
}

compare() {
	cd "$DIR"
	if [ ! -f latest.txt ]; then
		echo "No latest.txt to compare, run the benchmarks first" >&2
		exit 1
	fi
	if [ ! -f reference.txt ]; then
		echo "No reference.txt to compare with, use --update-reference to create it from latest.txt"
		return
	fi

	echo "Reference: $(grep -m 3 -E '^(commit|date|go):' reference.txt | tr '\n' ' ')"
	echo "Latest:    $(grep -m 3 -E '^(commit|date|go):' latest.txt | tr '\n' ' ')"
	echo

	# commit, date and go differ between reports, and would otherwise split them into separate tables
	go run "$BENCHSTAT" -ignore commit,date,go reference=reference.txt latest=latest.txt
	echo

	local color=0
	if [ -t 1 ]; then
		color=1
	fi

	go run "$BENCHSTAT" -ignore commit,date,go -format csv reference=reference.txt latest=latest.txt 2>/dev/null |
		awk -F, -v threshold="$THRESHOLD" -v color="$color" '
		# Lists changes benchstat found significant, splitting out those of at least threshold percent
		function paint(code, text) {
			return color ? "\033[" code "m" text "\033[0m" : text
		}

		/^,/ {
			if ($2 != "reference") {
				unit = $2
			}
			next
		}

		NF == 7 && $1 != "geomean" && $6 != "~" {
			name = $1
			sub(/-[0-9]+$/, "", name)
			change = $6
			size = threshold
			verdict = "changed"
			if (change ~ /%$/) {
				delta = substr(change, 1, length(change) - 1) + 0
				size = delta < 0 ? -delta : delta
				higherBetter = unit ~ /events\/s$/
				verdict = delta == 0 ? "same" : (higherBetter ? delta > 0 : delta < 0) ? "better" : "worse"
			}
			line = sprintf("%-60s %-15s %9s  %-7s  %s", name, unit, change, verdict, $7)
			if (size >= threshold) {
				large[++largeCount] = paint(verdict == "better" ? "1;32" : verdict == "worse" ? "1;31" : "1", line)
			} else {
				small[++smallCount] = paint("2", line)
			}
		}

		END {
			printf "Significant changes of at least %s%%:\n", threshold
			for (i = 1; i <= largeCount; i++) {
				print "  " large[i]
			}
			if (!largeCount) {
				print "  none"
			}
			printf "\nSmaller significant changes:\n"
			for (i = 1; i <= smallCount; i++) {
				print "  " small[i]
			}
			if (!smallCount) {
				print "  none"
			}
		}'
}

case "$1" in
"")
	run
	compare
	;;
--diff)
	compare
	;;
--update-reference)
	if [ ! -f "$DIR/latest.txt" ]; then
		echo "No latest.txt to use, run the benchmarks first" >&2
		exit 1
	fi
	cp "$DIR/latest.txt" "$DIR/reference.txt"
	echo "Reference updated from $(head -1 "$DIR/latest.txt")"
	;;
*)
	usage >&2
	exit 1
	;;
esac
