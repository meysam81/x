TEST_DIRS := "./ratelimit"

build:
  go build ./...

go-get:
  go get -u ./...

test:
  GOEXPERIMENT=synctest go test {{TEST_DIRS}}

test-full:
  GOEXPERIMENT=synctest go test -bench=. -benchmem {{TEST_DIRS}}

go-mod-tidy:
  for p in $(find . -maxdepth 2 -type f -name go.mod); do cd $(dirname $p) && go mod tidy && cd -; done

go-get-u:
  for p in $(find . -maxdepth 2 -type f -name go.mod); do cd $(dirname $p) && go get -u ./ && cd -; done
