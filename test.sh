#!/bin/bash


function generate_password {
	head /dev/urandom | LC_ALL=C tr -dc A-Za-z0-9 | head -c 20
}

function gcp_get_secret {
	./gcp-get-secret -verbose "$@"
}

function gcp_put_secret {
    local name value
    name=$1
    value=$2
    if ! gcloud --configuration integration-test secrets describe $name >/dev/null 2>&1 ; then
        gcloud --configuration integration-test secrets create $name --replication-policy automatic
    fi
    gcloud --configuration integration-test secrets versions add $name --data-file=- <<< "$value"
}

function assert_equals {
	if [[ $1 == $2 ]] ; then
		echo "INFO: ${FUNCNAME[1]} ok" >&2
	else
		echo "ERROR: ${FUNCNAME[1]} expected '$1'  got '$2'" >&2
	fi
}

function test_simple_get {
	local result expect
	expect=$(generate_password)
	gcp_put_secret mysql_root_password "$expect"
	result=$(gcp_get_secret --name mysql_root_password)
	assert_equals $expect $result
}




function test_get_via_env {
	local result expect
	expect=$(generate_password)
	gcp_put_secret mysql_root_password "$expect"
	result=$(MYSQL_PASSWORD=gcp:///mysql_root_password gcp_get_secret bash -c 'echo $MYSQL_PASSWORD')
	assert_equals $expect $result
}

function test_get_via_env_default {
	local result expect
	expect=$(generate_password)
	result=$(MYSQL_PASSWORD="gcp:///there-is-no-such-parameter-in-the-store-is-there?default=$expect" gcp_get_secret bash -c 'echo $MYSQL_PASSWORD')
	assert_equals $expect $result
}

function test_template_format {
	local result expect password
	password=$(generate_password)
	gcp_put_secret postgres_root_password "$password"
	expect="localhost:5432:postgres:root:${password}"

	result=$(TMP=/tmp \
	         PGPASSFILE=gcp:///postgres_root_password?template='localhost:5432:postgres:root:{{.}}%0A&destination=$TMP/.pgpass' \
		gcp_get_secret bash -c 'cat $PGPASSFILE')
	assert_equals $expect $result
}

function test_env_substitution {
	local result expect
	expect=$(generate_password)
	gcp_put_secret ${expect}_mysql_root_password "$expect"
	result=$(ENV=$expect \
                PASSWORD='gcp:///${ENV}_mysql_root_password' \
	        gcp_get_secret bash -c 'echo $PASSWORD')
	assert_equals $expect $result
}

function test_destination {
	local result expect filename
	expect=$(generate_password)
	filename=/tmp/password-$$
	gcp_put_secret postgres_root_password "$expect"

	result=$(FILENAME=$filename \
	         PASSWORD_FILE='gcp:///postgres_root_password?destination=$FILENAME&chmod=0600' \
		gcp_get_secret bash -c 'echo $PASSWORD_FILE')
	assert_equals $filename $result
	assert_equals $expect $(<$filename)
	assert_equals 600 $(stat -f %A $filename)
	rm $filename
}

function test_umask {
	local result expect filename
	expect=$(generate_password)
	filename=/tmp/password-$$
	gcp_put_secret postgres_root_password "$expect"

	result=$(FILENAME=$filename \
	         PASSWORD_FILE='gcp:///postgres_root_password?destination=$FILENAME' \
		gcp_get_secret -umask 0077 bash -c 'echo $PASSWORD_FILE')
	assert_equals $filename $result
	assert_equals $expect $(<$filename)
	assert_equals 600 $(stat -f %A $filename)
	rm $filename
}


function test_destination_directory_creation {
	local result expect filename
	expect=$(generate_password)
	filename=$(mktemp -d)/create-this-directory-here/password-$$
	gcp_put_secret postgres_root_password "$expect"

	result=$(FILENAME=$filename \
	         PASSWORD_FILE='gcp:///postgres_root_password?destination=$FILENAME&chmod=0600' \
		gcp_get_secret bash -c 'echo $PASSWORD_FILE')
	assert_equals $filename $result
	assert_equals $expect $(<$filename)
	assert_equals 600 $(stat -f %A $filename)
	rm $filename
}

function test_destination_default {
	local result expect filename
	expect=$(generate_password)
	filename=/tmp/password-$$
	echo -n "$expect" > $filename
	result=$(FILENAME=$filename \
	         PASSWORD_FILE='gcp:///there-is-no-such-parameter-in-the-store-is-there?destination=$FILENAME&chmod=0600' \
		gcp_get_secret bash -c 'echo $PASSWORD_FILE')
	assert_equals $filename $result
	assert_equals $expect $(<$filename)
	assert_equals 600 $(stat -f %A $filename)
	rm $filename
}

function test_full_secret_name {
	local result expect
	expect=$(generate_password)
	gcp_put_secret mysql_root_password "$expect"
	result=$(gcp_get_secret --name mysql_root_password)
	assert_equals $expect $result

    name=$(gcloud --configuration integration-test secrets versions list mysql_root_password  --format="json" | jq -r '.[0].name')
    result=$(gcp_get_secret --name $name)
}


function test_shorthand_version {
	local result expect
	expect=$(generate_password)
	gcp_put_secret mysql_root_password "$expect"
    name=$(gcloud --configuration integration-test secrets versions list mysql_root_password  --format="json" | jq -r '.[0].name')
    version=$(sed -e s'~.*/versions/\(.*\)~\1~' <<< "$name")
    # add a new version, so we get the previous one
    gcp_put_secret mysql_root_password "$(generate_password)"
    result=$(gcp_get_secret --name mysql_root_password/$version)
    assert_equals $expect $result
}

function test_shorthand_project {
	local result expect
	expect=$(generate_password)
	gcp_put_secret mysql_root_password "$expect"
    name=$(gcloud --configuration integration-test secrets versions list mysql_root_password  --format="json" | jq -r '.[0].name')

    project=$(sed -e s'~projects/\([^/]*\)/.*~\1~' <<< "$name")
    result=$(gcp_get_secret --name $project/mysql_root_password)
    assert_equals $expect $result
}


function test_shorthand_project_and_version {
	local result expect
	expect=$(generate_password)
	gcp_put_secret mysql_root_password "$expect"
    name=$(gcloud --configuration integration-test secrets versions list mysql_root_password  --format="json" | jq -r '.[0].name')

    project=$(sed -e s'~projects/\([^/]*\)/.*~\1~' <<< "$name")
    version=$(sed -e s'~.*/versions/\(.*\)~\1~' <<< "$name")

    result=$(gcp_get_secret --name $project/mysql_root_password/$version)
    assert_equals $expect $result
}


function main {
    gcloud config configurations activate integration-test
    go build -o gcp-get-secret .
    test_umask
    test_shorthand_project_and_version
    test_shorthand_project
    test_shorthand_version
    test_full_secret_name
    test_simple_get
    test_get_via_env
    test_get_via_env_default
    test_destination
    test_destination_default
    test_destination_directory_creation
    test_env_substitution
    test_template_format
}

main
