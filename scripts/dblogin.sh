set -euo pipefail

ESCAPED_DB_PASS=$(echo -n ${DB_PASS} | hexdump -v -e '/1 "%02x"' | sed 's/\(..\)/%\1/g')
PGPASSWORD="$DB_PASS" psql -h $DB_HOST main ${DB_USER}@${DB_HOST}

