.PHONY: help build deploy status logs restart

help:
	@echo "Available commands:"
	@echo "  make build   - Build the Go binary locally"
	@echo "  make deploy  - Deploy to EC2 (pull, build, restart)"
	@echo "  make status  - Check service status on EC2"
	@echo "  make logs    - Show service logs on EC2"
	@echo "  make restart - Restart service on EC2"

build:
	go build -o cupid main.go

deploy:
	ssh cupid-bot "source ~/.bash_profile && cd ~/cupid && git pull && go build -o cupid main.go && sudo systemctl restart cupid && sudo systemctl status cupid"

status:
	ssh cupid-bot "sudo systemctl status cupid"

logs:
	ssh cupid-bot "sudo journalctl -u cupid -n 50 --no-pager"

restart:
	ssh cupid-bot "sudo systemctl restart cupid && sudo systemctl status cupid"
