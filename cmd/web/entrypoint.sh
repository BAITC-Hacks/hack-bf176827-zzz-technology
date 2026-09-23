#!/bin/sh
set -eu
/app/bin/pipeline --data "${APP_DATA_DIR:-data}" --out "${APP_OUT_DIR:-out}"
/app/bin/check --out "${APP_OUT_DIR:-out}"
exec /app/bin/web
