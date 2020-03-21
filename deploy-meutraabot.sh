#!/usr/bin/env sh
go build ./cmd/meutraabot || exit

rm meutraabot meutraa-leaderboard

# lines=$(sloc -f csv cmd/meutraabot/main.go | tail -1 | cut -d',' -f3)

# sed -i "s/lines of code\", [^)]*/lines of code\", ${lines}/" cmd/meutraabot/main.go

rsync -aP -e 'ssh -p 2020' --delete ./ lost.host:meutraabot/
ssh -p 2020 lost.host /usr/bin/env sh << EOF
cd meutraabot
go build ./cmd/meutraabot && \
        sudo machinectl shell root@services /bin/sh -c 'systemctl stop meutraabot' && \
        sleep 1s && \
        sudo cp meutraabot /var/lib/containers/services/etc/nixos/bin/ && \
        sudo machinectl shell root@services /bin/sh -c 'systemctl start meutraabot' && \
        sudo machinectl shell root@services /bin/sh -c 'journalctl -xefu meutraabot'
EOF
