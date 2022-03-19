# Copyright © 2022 Michael Thompson
# SPDX-License-Identifier: GPL-2.0-or-later

BINDIR=bin

build:
	go build -o "${BINDIR}/layercake" ./cmd/layercake/...
	go build -o "${BINDIR}/stagemaker" ./cmd/stagemaker/...


test:
	go test ./...

