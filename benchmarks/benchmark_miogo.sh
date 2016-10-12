#!/bin/bash

case "$(pidof miogo | wc -w)" in
    0)  echo "launch miogo and launch go test before running the benchmark "
        exit 1
        ;;
    1)  echo "launch go test before the benchmark"
        ;;
esac

curl -c session.txt --silent --data "email=test@miogo.tld&password=test" http://localhost:8080/Login > /dev/null

session=$(cat session.txt | grep session | cut -f7)

cat >benchmark_miogo.lua <<EOF
wrk.method = "POST"
wrk.body   = "path=/test/README.md"
wrk.headers["Content-Type"] = "application/x-www-form-urlencoded"
wrk.headers["Cookie"] = "session=$session"
EOF

wrk -v -t2 -c10 -d30s -s ./benchmark_miogo.lua http://localhost:8080/GetFile


