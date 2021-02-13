module github.com/edgebox-iot/sysctl

go 1.15

replace github.com/edgebox-iot/sysctl/internal/database => ./internal/database

require (
	github.com/edgebox-iot/sysctl/internal/database v0.0.0-00010101000000-000000000000
	github.com/go-sql-driver/mysql v1.5.0 // indirect
)
