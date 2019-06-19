export GOPATH=$(pwd)/../../..

go build -i -o ../tools/code_generator github.com/huoshan017/mysql-go/code_generator
go build -i -o ../tools/db_proxy_server github.com/huoshan017/mysql-go/proxy/server