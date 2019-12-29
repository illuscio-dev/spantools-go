.PHONY: test
test:
	-go test -v -failfast -covermode=count -coverprofile=zdevelop/tests/_reports/coverage.out -coverpkg=./... ./...
	-go tool cover -html=zdevelop/tests/_reports/coverage.out
	open ./zdevelop/tests/zreports/coverage.html

.PHONY: lint
lint:
	-revive -config revive.toml ./...

.PHONY: format
format:
	-gofmt -s -w ./
	-gofmt -s -w ./zdevelop/tests

.PHONY: venv
venv:
ifeq ($(py), )
	$(eval PY_PATH := python3)
else
	$(eval PY_PATH := $(py))
endif
	$(eval VENV_PATH := $(shell $(PY_PATH) ./zdevelop/make_scripts/make_venv.py))
	@echo "venv created! To enter virtual env, run:"
	@echo ". ~/.bash_profile"
	@echo "then run:"
	@echo "$(VENV_PATH)"

.PHONY: install-dev
install-dev:
	pip install --upgrade pip
	pip install -r requirements.txt

.PHONY: doc
doc:
	docmodule-go
	python setup.py build_sphinx -E
	sleep 1
	open ./zdocs/build/html/index.html
