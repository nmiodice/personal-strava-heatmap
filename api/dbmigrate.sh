set -euo pipefail

echo "escaping psql-pass"
ESCAPED_DB_PASS=$(echo -n ${DB_PASS} | hexdump -v -e '/1 "%02x"' | sed 's/\(..\)/%\1/g')

echo "running migrate command"
echo "${1:-up}" "${@:2}"
migrate -database "postgres://${DB_USER}@${DB_HOST}:${ESCAPED_DB_PASS}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE}" -path migrations "${1:-up}" "${@:2}"
