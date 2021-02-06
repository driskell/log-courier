# Testing

## Running ES From Docker

    docker pull docker.elastic.co/elasticsearch/elasticsearch:7.10.2
    docker run -p 9200:9200 -p 9300:9300 -e "discovery.type=single-node" docker.elastic.co/elasticsearch/elasticsearch:7.10.2

## Generate Log Files

    for FILE in multiline singleline; do
      COUNT=0
      exec 3>"$FILE-generated.log"
      DATA=$(cat "$FILE.log")
      while (( COUNT < 1000000 )); do
        echo "$DATA" >&3
        (( COUNT++ ))
      done
      exec 3>&-
    done

## Starting Log Carver

    log-carver -config log-carver.yaml -config-debug

## Starting Log Courier

    log-courier -config log-courier.yaml -config-debug

## Resetting Log Courier Resume

    rm -f .log-courier
