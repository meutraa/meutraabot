#!/usr/bin/env sh
go build ./cmd/meutraabot || exit

rm meutraabot

rsync -aP --delete ./ lost:meutraabot/
ssh lost /usr/bin/env sh << EOF
cd meutraabot
go build ./cmd/meutraabot && \
        sudo systemctl stop meutraabot && \
        sudo cp meutraabot /etc/nixos/bin/ && \
        sudo systemctl start meutraabot && \
        sudo journalctl -xefu meutraabot
EOF
