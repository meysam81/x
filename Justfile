TEST_DIRS := "./ratelimit"

build:
  go build ./...

go-get:
  go get -u ./...

test:
  GOEXPERIMENT=synctest go test {{TEST_DIRS}}

test-full:
  GOEXPERIMENT=synctest go test -bench=. -benchmem {{TEST_DIRS}}
