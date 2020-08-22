#!/usr/bin/env sh
go build ./cmd/meutraabot || exit

rm meutraabot

rsync -aP --delete ./ lost:meutraabot/
ssh lost /usr/bin/env sh << EOF
cd meutraabot
CGO_ENABLED=0 go build -ldflags "-extldflags -static" ./cmd/meutraabot && \
        sudo systemctl stop meutraabot && \
        sudo cp meutraabot /etc/nixos/bin/meutraabot/ && \
        sudo systemctl start meutraabot && \
        sudo journalctl -xefu meutraabot
EOF
