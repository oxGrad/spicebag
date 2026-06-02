build-frontend:
  cd frontend && npm ci && npm run build

build: build-frontend
  go build -o spicebag ./cmd/spicebag/

run: build
  ./spicebag start

test:
  go test ./...

clean:
  rm -f ./spicebag
  rm -rf internal/dashboard/ui/
