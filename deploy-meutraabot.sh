#!/usr/bin/env sh
go build ./cmd/meutraabot || exit

rm meutraabot meutraa-leaderboard

# lines=$(sloc -f csv cmd/meutraabot pkg/data pkg/irc | tail -1 | cut -d',' -f3)

# sed -i "s/lines of code\", [^)]*/lines of code\", ${lines}/" cmd/meutraabot/main.go

rsync -aP -e 'ssh -p 2020' --delete ./ lost.host:meutraabot/
ssh -p 2020 lost.host /usr/bin/env sh << EOF
cd meutraabot
go build ./cmd/meutraabot && \
        sudo systemctl stop meutraabot && \
        sudo cp meutraabot /etc/nixos/bin/ && \
        sudo systemctl start meutraabot && \
        sudo journalctl -xefu meutraabot
EOF
