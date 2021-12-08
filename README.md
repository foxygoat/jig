# Pony

Because everybody wants a pony.

## Playing

Build and start the echo gRPC server with

	. ./bin/activate-hermit
	make install
	server

in a second terminal call it with

	client

To see streaming, call it with

        client --stream=server
        client --stream=client you me world
        client --stream=bidi you me world

## Development

	. ./bin/activate-hermit
	make ci

Run `make help` for help on other make targets.
