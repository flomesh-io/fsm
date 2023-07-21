#!/bin/bash

TEMP_DIR=$(mktemp -d)
tar -C ${TEMP_DIR} -zxf ${SCRIPTS_TAR}

RET=0
diff -qr ${TEMP_DIR}/scripts ${CHART_COMPONENTS_DIR}/scripts || RET=$?

if [[ ${RET} -eq 0 ]]
then
  echo "${SCRIPTS_TAR} is up to date."
else
    echo -e "\nPlease commit the changes made by 'make package-scripts'"
    exit 1
fi
