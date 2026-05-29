#!/usr/bin/env sh
set -eu

secure_path="${OUUI_SECURE_PATH:-/ou-ui}"
enable_ssl="${OUUI_ENABLE_SSL:-no}"
cert_file="${OUUI_TLS_CERT_FILE:-/app/certs/fullchain.pem}"
key_file="${OUUI_TLS_KEY_FILE:-/app/certs/privkey.pem}"

case "$secure_path" in
  /*) ;;
  *) secure_path="/$secure_path" ;;
esac

listen_line="listen 3000;"
ssl_lines=""

if [ "$enable_ssl" = "yes" ]; then
  if [ ! -s "$cert_file" ] || [ ! -s "$key_file" ]; then
    echo "OU-UI HTTPS is enabled, but certificate files are missing." >&2
    echo "Expected: $cert_file and $key_file" >&2
    exit 1
  fi
  listen_line="listen 3000 ssl;"
  ssl_lines="
  ssl_certificate $cert_file;
  ssl_certificate_key $key_file;
  ssl_protocols TLSv1.2 TLSv1.3;
  ssl_session_cache shared:SSL:10m;"
fi

cat >/etc/nginx/conf.d/default.conf <<EOF_NGINX
server {
  $listen_line
  server_name _;
  root /usr/share/nginx/html;
  index index.html;
$ssl_lines

  location = /healthz {
    proxy_pass http://ou-ui-server:8080/healthz;
  }

  location ${secure_path}/api/ {
    proxy_pass http://ou-ui-server:8080${secure_path}/api/;
    proxy_set_header Host \$host;
    proxy_set_header X-Real-IP \$remote_addr;
    proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto \$scheme;
  }

  location ${secure_path}/healthz {
    proxy_pass http://ou-ui-server:8080${secure_path}/healthz;
  }

  location ${secure_path}/ {
    rewrite ^${secure_path}/(.*)\$ /\$1 break;
    try_files \$uri \$uri/ /index.html;
  }

  location = ${secure_path} {
    return 302 ${secure_path}/;
  }

  location / {
    return 404;
  }
}
EOF_NGINX
