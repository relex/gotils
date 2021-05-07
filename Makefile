GOPATH := $(shell go env GOPATH)

# print help by default 
.PHONY: help
help:
	@echo "Run \"make install\" to install scripts to ${GOPATH}"
	@echo "Run \"make uninstall\" to unintall scripts from ${GOPATH}"
	@echo ""
	@echo "Run \"make test\" to test child libraries (after \"make install\")"
	@echo "Run \"make lint\" to check all code (after \"make install\")"
	@echo "Run \"make pretty\" to reformat all code"
	@echo "Run \"make upgrade\" to upgrade dependencies"
	@echo ""
	@go version

.PHONY: install
install:
	mkdir -p ${GOPATH}/bin
	install scripts/gotils-prereq.sh ${GOPATH}/bin/
	install scripts/gotils-build.sh ${GOPATH}/bin/
	install scripts/gotils-lint.sh ${GOPATH}/bin/
	install scripts/gotils-test.sh ${GOPATH}/bin/
	mkdir -p ${GOPATH}/opt/gotils
	cp -a templates ${GOPATH}/opt/gotils/
	cp -a Common.mk ${GOPATH}/opt/gotils/
	cp -a README.md ${GOPATH}/opt/gotils/
	${GOPATH}/bin/gotils-prereq.sh

.PHONY: uninstall
uninstall:
	rm -f ${GOPATH}/bin/gotils-*.sh
	rm -fr ${GOPATH}/opt/gotils

include Common.mk
