#!/usr/bin/env sh
go build ./cmd/meutraa-leaderboard || exit

rm meutraabot meutraa-leaderboard meutraa-chat

rsync -aP -e 'ssh -p 2020' --delete ./ lost.host:meutraabot/
ssh -p 2020 lost.host /usr/bin/env sh << EOF
cd meutraabot
go build ./cmd/meutraa-leaderboard && \
        sudo systemctl stop meutraa-leaderboard && \
        sudo cp meutraa-leaderboard /etc/nixos/bin/ && \
        sudo systemctl start meutraa-leaderboard && \
        sudo journalctl -xefu meutraa-leaderboard
EOF
