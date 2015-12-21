#!/bin/bash
set -e

pushd `dirname $0` > /dev/null
dir=`pwd`
popd > /dev/null

trap "echo cleanup; rm -Rf ${dir}/tests/*/target/; exit" EXIT HUP INT QUIT PIPE TERM


if [ "$(id -u)" != "0" ]; then
	echo "Sorry, you are not root."
	exit 1
fi

export CNT_PATH="${dir}/dist/linux-amd64/cnt"

command -v rkt >/dev/null 2>&1 || { echo >&2 "rkt not found in path"; exit 1; }
command -v bats >/dev/null 2>&1 || { echo >&2 "bats not found in path"; exit 1; }
command -v ${CNT_PATH} >/dev/null 2>&1 || { echo >&2 "build cnt first"; exit 1; }

execute_tests() {
  fdir=$1
  [ -d "$fdir" ] || return 0

  for file in $fdir/*; do
    filename=$(basename $file)
    [ "$filename" == "wait.sh" ] && continue
    [ -d "$file" ] && continue

    echo -e "\e[1m\e[32mRunning test file -> $filename\e[0m"
    bats -p $file
  done
}

cd "${dir}/tests"
execute_tests "."