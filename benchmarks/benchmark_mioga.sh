#!/bin/bash

curl -c session.txt --silent --data "login=root%40localhost.tld&password=admin" http://localhost/Mioga2/login/DisplayMain?target=/Mioga2/Mioga/bin/admin/Workspace/DisplayMain

session=$(cat session.txt | grep session | cut -f7)

cat >benchmark_mioga.lua <<EOF
wrk.headers["Cookie"] = "mioga_session=$session"
EOF

wrk -v -t2 -c10 -d30s -s ./benchmark_mioga.lua http://localhost/Mioga2/Mioga/home/admin/README.md
