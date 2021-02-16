module github.com/edgebox-iot/sysctl

go 1.15

replace github.com/edgebix-iot/sysctl/internal/edgeapps => ./internal/edgeapps

replace github.com/edgebox-iot/sysctl/internal/utils => ./internal/utils

replace github.com/edgebox-iot/sysctl/internal/tasks => ./internal/tasks

require (
	github.com/edgebox-iot/sysctl/internal/tasks v0.0.0-00010101000000-000000000000
	github.com/edgebox-iot/sysctl/internal/utils v0.0.0-00010101000000-000000000000 // indirect
	github.com/go-sql-driver/mysql v1.5.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
