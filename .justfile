build-frontend:
  cd frontend && npm ci && npm run build

build-frontend-dev:
  cd frontend && npm run build

build: build-frontend
  go build -o spicebag ./cmd/spicebag/

build-go:
  go build -o spicebag ./cmd/spicebag/

dev: build-frontend-dev build-go
  ./spicebag start

run: build
  ./spicebag start

test:
  go test ./...

clean:
  rm -f ./spicebag
  rm -rf internal/dashboard/ui/
