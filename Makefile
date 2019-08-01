GIT_SHA=$(shell git rev-parse HEAD)

.PHONY: increment_version
increment_version:
	head -n -1 version.go > /tmp/version.go; mv /tmp/version.go version.go
	echo 'const currentVersion = "$(GIT_SHA)"' >> version.go

test:
	bash .envrc && go test
