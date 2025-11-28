#!/bin/bash

# Script para ejecutar tests con las variables de entorno correctas
# Basado en .github/workflows/mergeintoalpha.yml

# Variables de entorno requeridas por los tests (igual que en CI)
# IMPORTANTE: Forzar override de variables del usuario
export WST_ADMIN_USERNAME="admin"
export WST_ADMIN_PWD="Abcd1234."
export PPROF_AUTH_USERNAME="${PPROF_AUTH_USERNAME:-test}"
export PPROF_AUTH_PASSWORD="${PPROF_AUTH_PASSWORD:-abcd1234.}"
export MINIO_ACCESS_KEY="${MINIO_ACCESS_KEY:-8BpW4Hd8bWpJS4rVu1oq}"
export MINIO_DOMAIN="${MINIO_DOMAIN:-minio.gacodes.com}"
export MINIO_SECRET_KEY="${MINIO_SECRET_KEY:-FPdiJ3ut4hRjaUJsgxd9XLKr8wYwbyi5nhvzNG61}"
export MIN_COVERAGE_THRESHOLD="${MIN_COVERAGE_THRESHOLD:-81.3}"
export JWT_SECRET="${JWT_SECRET:-abcD12345678.}"
export GO_ENV="${GO_ENV:-TESTING}"
export DEBUG="${DEBUG:-true}"
export PORT="${PORT:-8019}"

# IMPORTANTE: Eliminar variables de MongoDB que interfieren con tests locales
unset WST_DB0_USERNAME
unset WST_DB0_PASSWORD
unset WST_DB1_USERNAME
unset WST_DB1_PASSWORD
unset WST_DB2_USERNAME
unset WST_DB2_PASSWORD
unset WST_DB_EXPECTED_TO_FAIL_USERNAME
unset WST_DB_EXPECTED_TO_FAIL_PASSWORD
unset WST_DB_EXPECTED_TO_BE_CLOSED_USERNAME
unset WST_DB_EXPECTED_TO_BE_CLOSED_PASSWORD

# Verificar que MongoDB esté corriendo
if ! docker ps | grep -q mongodb; then
    echo "⚠️  MongoDB no está corriendo. Iniciando MongoDB sin autenticación..."
    docker run -d --name mongodb --rm -p 27017:27017 mongo --noauth
    echo "⏳ Esperando 3 segundos para que MongoDB inicie..."
    sleep 3
fi

# Ejecutar los tests
echo "🧪 Ejecutando tests..."
go test -v -timeout 600s "$@" ./tests/

# Capturar el código de salida
TEST_EXIT_CODE=$?

# Mostrar resumen
echo ""
echo "════════════════════════════════════════════════"
if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo "✅ Tests completados exitosamente"
else
    echo "❌ Tests fallaron con código: $TEST_EXIT_CODE"
fi
echo "════════════════════════════════════════════════"

exit $TEST_EXIT_CODE
