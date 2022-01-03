#!/bin/sh

# Install deps
apk add postgresql-client curl jq

echo "Provided connection string: '${PSQL_URL}'"

# Extract the database name from the end of the PSQL URL, and check it's there
DB_PARAMS=`echo ${PSQL_URL} | sed 's!^.*/!!'`
DB_NAME=`echo ${DB_PARAMS} | sed 's!?.*!!'`
echo "Database name: '${DB_NAME}'"
USER_NAME=`echo ${PSQL_URL} | sed 's!^.*//!!' | sed 's!:.*$!!'`
echo "Username: '${USER_NAME}'"
COLONS=`echo -n $DB_NAME | sed 's/[^:]//g'`
if [ -z "${DB_NAME}" ] || [ -n "${COLONS}" ]
then
  echo "Error: Postgres URL does not appear to contain a database name (required)."
  exit 1
fi

# Check we can connect to the PSQL server using the default "postgres" database
PSQL_SERVER=`echo ${PSQL_URL} | sed "s!${DB_PARAMS}!!"`postgres
echo "PSQL server URL: '${PSQL_SERVER}'"
until psql -c "SELECT 1;" ${PSQL_SERVER}; do
  echo "Waiting for PSQL server connection..."
  sleep 1
done

# Create the database if it doesn't exist
if ! psql -c "SELECT datname FROM pg_database WHERE datname = '${DB_NAME}';" ${PSQL_SERVER} | grep ${DB_NAME}
then
  echo "Database '${DB_NAME}' does not exist; creating."
  psql -c "CREATE DATABASE \"${DB_NAME}\" WITH OWNER \"${USER_NAME}\";" ${PSQL_SERVER}
fi

# Wait for the database itself to be available
until psql -c "SELECT 1;" ${PSQL_URL}; do
  echo "Waiting for database..."
  sleep 1
done

# Download the latest migration tool
MIGRATE_RELEASE=$(curl -sL https://api.github.com/repos/golang-migrate/migrate/releases/latest | jq -r '.name')
curl -sL https://github.com/golang-migrate/migrate/releases/download/${MIGRATE_RELEASE}/migrate.linux-amd64.tar.gz | tar xz

# Do the migrations
./migrate -database ${PSQL_URL} -path db/migrations/postgres up
