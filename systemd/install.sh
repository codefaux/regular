#! /bin/sh
set -eu

systemd_dir=/etc/systemd/system
service_file=regular.service

cd "$(dirname "$0")"

echo "Building regular with go"
go build ..

echo "Installing regular to /usr/bin/regular"
install regular /usr/bin/regular

echo "Installing regular.service to /etc/systemd/system"
install "$service_file" "${systemd_dir}/${service_file}"

echo "Reloading systemd, enabling and starting regular"
systemctl daemon-reload
systemctl enable "$service_file" --now
