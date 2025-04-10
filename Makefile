# Help is printed out based on the comment following the command name.
help: # Show help for each of the Makefile targets.
	@grep -E '^[a-zA-Z0-9 -]+:.*#'  Makefile | sort | while read -r l; do printf "\033[1;32m$$(echo $$l | cut -f 1 -d':')\033[00m:$$(echo $$l | cut -f 2- -d'#')\n"; done

up: # Start the service using docker compose.
	docker compose up

down: # Stop the service started with up.
	docker compose down

test-unit: # Run unit test
	go test -race ./...
