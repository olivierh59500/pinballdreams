.PHONY: run build test vet android android-run

run:
	./scripts/run.sh

build:
	./scripts/build.sh

test:
	GOCACHE="$(CURDIR)/.cache/go-build" go test ./...

vet:
	GOCACHE="$(CURDIR)/.cache/go-build" go vet ./...

android:
	./scripts/build-android.sh

android-run:
	./scripts/run-android.sh
