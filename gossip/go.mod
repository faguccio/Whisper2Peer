module gossip

go 1.22.3

require (
	github.com/lmittmann/tint v1.0.4
	github.com/neilotoole/slogt v1.1.0
	gopkg.in/ini.v1 v1.67.0
)

require (
	github.com/stretchr/testify v1.9.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
)

replace capnproto.org/go/capnp/v3 => github.com/capnproto/go-capnp/v3 v3.0.0-alpha.30.0.20240430165919-f68dd6e12692

require capnproto.org/go/capnp/v3 v3.0.0-00010101000000-000000000000
