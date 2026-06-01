build:
  go build -o spicebag ./cmd/spicebag/

test:
  go test ./...

clean:
  rm -f ./spicebag
