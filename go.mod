module github.com/antibubblewrap/tradekit

go 1.20

require (
	github.com/edwingeng/deque/v2 v2.1.1
	github.com/gorilla/websocket v1.5.0
	github.com/stretchr/testify v1.8.4
	github.com/valyala/fastjson v1.6.4
	golang.org/x/exp v0.0.0-20230810033253-352e893a4cad
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

retract (
	v0.3.0
	v0.2.0
	v0.1.0
)
