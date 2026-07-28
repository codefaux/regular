#! /bin/sh
set -eu

cd "$(dirname "$0")"

go build ..
install regular /usr/bin/regular

systemd_dir=/etc/systemd/system
service_file=regular.service

mkdir -p "$systemd_dir"
install "$service_file" "${systemd_dir}/${service_file}"

systemctl daemon-reload
systemctl enable "$service_file"
systemctl start "$service_file"
