module github.com/steampoweredtaco/gotiktoklive

go 1.23.0

require (
	github.com/gobwas/ws v1.1.0
	github.com/stretchr/testify v1.9.0
	go.uber.org/ratelimit v0.3.1
)

retract (
	v1.0.8 // retration only update.
	v1.0.7 // Published accidently.
)

require (
	github.com/benbjohnson/clock v1.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require (
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/pkg/errors v0.9.1
	golang.org/x/net v0.25.0
	golang.org/x/sys v0.20.0 // indirect
	google.golang.org/protobuf v1.33.0
)
