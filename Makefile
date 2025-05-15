# Copyright © 2022 Michael Thompson
# SPDX-License-Identifier: GPL-2.0-or-later

PREFIX ?= /usr
BINDIR ?= $(PREFIX)/bin
MAN1DIR ?= $(PREFIX)/share/man/man1
INFODIR ?= $(PREFIX)/share/info

executables = layercake stagemaker
man_files = layercake.1 stagemaker.1
info_files = layercake.info

bin: $(addprefix bin/, $(executables))

bin/layercake:
	CGO_ENABLED=0 go build -o ./bin/layercake ./cmd/layercake/...

bin/stagemaker:
	CGO_ENABLED=0 go build -o ./bin/stagemaker ./cmd/stagemaker/...

man: $(addprefix doc/, $(man_files))

%.1: %_manpage.adoc
	a2x -f manpage $<

doc/layercake.info: doc/layercake.adoc
	asciidoc -b docbook -d book -a data-uri -o doc/layercake.xml doc/layercake.adoc
	docbook2x-texi doc/layercake.xml --encoding=UTF-8 --to-stdout >doc/layercake.texi
	makeinfo --no-split -o doc/layercake.info doc/layercake.texi

info: doc/layercake.info

install-bin: $(addprefix bin/, $(executables)) | $(DESTDIR)$(BINDIR)
	install -m 755 $(addprefix bin/, $(executables)) $(DESTDIR)$(BINDIR)

install-man: $(addprefix doc/, $(man_files)) | $(DESTDIR)$(MAN1DIR)
	install -m 644 $(addprefix doc/, $(man_files)) $(DESTDIR)$(MAN1DIR)

install-info: $(addprefix doc/, $(info_files)) | $(DESTDIR)$(INFODIR)
	install -m 644 $(addprefix doc/, $(info_files)) $(DESTDIR)$(INFODIR)

$(DESTDIR)$(BINDIR) $(DESTDIR)$(MAN1DIR) $(DESTDIR)$(INFODIR):
	install -d -m 755 $@

install: install-bin install-man install-info

dist: bin man info
	go run makedist.go

clean:
	rm -rf dist bin/*
	rm doc/\*.{html,info,man,texi,xml}

test:
	go test ./...


.PHONY: bin man info install-bin install-man install-info install dist clean test

