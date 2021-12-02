# Pony

Because everybody wants a pony.

## Playing

Build and start the echo gRPC server with

	. ./bin/activate-hermit
	make install
	server

in a second terminal call it with

	client

Instead of `server` try starting `dynamicserver` to run a gRPC server
without code generation.

## Development

	. ./bin/activate-hermit
	make ci

Run `make help` for help on other make targets.
