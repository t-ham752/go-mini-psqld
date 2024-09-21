# go-mini-psqld
The library is helpful for building applications that can connect via psql. By integrating it with your own database, you can connect to it from psql command.

this library is inspired by https://github.com/goropikari/psqlittle.

## Usage

### Build tcp server
```go
const serverVersion = "14.11 (Debian 14.11-1.pgdg110+2)"

func main() {
	conf := &server.TCPServerConfig{
		Port:         54322,
		QueryHandler: queryHandler,
	}
	server := server.NewTCPServer(conf,
		server.WithServerVersion(serverVersion),
		server.WithTimeZone("Asia/Tokyo"),
	)

	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("failed to start server: %+v", err)
		}
	}()

	quit := make(chan struct{})
	<-quit
}

func queryHandler(query []byte) ([]byte, error) {
    // you can parse query and return response
}
```

### Connect via psql
```shell
$ psql -U postgres -h 127.0.0.1 -p 54322
psql (15.8 (Homebrew)、server 14.11 (Debian 14.11-1.pgdg110+2))

postgres=> 
```
