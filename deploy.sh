go build -ldflags "-s -w" ./cmd/meutraabot
scp meutraabot root@moon:
ssh root@moon "
        systemctl stop meutraabot
	cp meutraabot /usr/local/bin/
	chcon -R -u system_u -r object_r -t bin_t /usr/local/bin/meutraabot
	systemctl start meutraabot
	journalctl --since=\"1 second ago\" -xefu meutraabot
"
