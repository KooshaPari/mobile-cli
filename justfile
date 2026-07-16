# mobile-cli — Build system alias (just = make replacement)
set dotenv-load

# default: list recipes
default:
    @just --list

# install
install:
    @echo "TODO: install mobile-cli deps"

# build
build:
    @echo "TODO: build mobile-cli"

# test
test:
    @echo "TODO: test mobile-cli"

# lint
lint:
    @echo "TODO: lint mobile-cli"

# format
format:
    @echo "TODO: format mobile-cli"

# verify (justfile-verify-in-pre-commit hook gate)
verify:
    @just --evaluate
