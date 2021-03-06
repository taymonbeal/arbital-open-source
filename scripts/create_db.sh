#!/bin/bash
#
# Creates xelaie DB and tables on MySQL server at localhost.

source init.sh || exit

HOST=localhost

read -r -p "This script will DROP ALL DB DATA and rebuild the database at ${HOST}. Is this your intent? [y/N] " response
if [[ ! $response =~ ^([yY][eE][sS]|[yY])$ ]]; then
	 exit
fi

DB_NAME=$(cfg mysql.database)
DB_USER=$(cfg mysql.user)
ROOT_PW=$(cfg mysql.root.password)
USER_PW=$(cfg mysql.password)

echo "Creating DB ${DB_NAME}@${HOST}.."
mysql --host ${HOST} -u root -p"${ROOT_PW}" -e "DROP DATABASE IF EXISTS ${DB_NAME}; CREATE DATABASE IF NOT EXISTS ${DB_NAME} DEFAULT CHARACTER SET utf8mb4 DEFAULT COLLATE utf8mb4_general_ci; USE ${DB_NAME};"

echo "Creating user ${DB_USER}.."
# Note that "GRANT" also creates the user, if necessary (no point in using "CREATE USER"):
# http://justcheckingonall.wordpress.com/2011/07/31/create-user-if-not-exists/
mysql --host ${HOST} -u root -p"${ROOT_PW}" -e "GRANT ALL ON ${DB_NAME}.* TO '${DB_USER}'@'%' IDENTIFIED BY '${USER_PW}';"

SCHEMAS=schemas/*.sql
for f in $SCHEMAS; do
	echo "Importing schema ${f}.."
	cat ${f} | mysql --host ${HOST} -u ${DB_USER} -p${USER_PW} ${DB_NAME}
done

echo "All done."
