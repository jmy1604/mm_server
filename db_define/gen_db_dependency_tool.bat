set GOPATH=%cd%/../../../

go build -i -o ../tools/code_generator.exe github.com/huoshan017/mysql-go/code_generator
go build -i -o ../tools/db_proxy_server.exe github.com/huoshan017/mysql-go/proxy/server