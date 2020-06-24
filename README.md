# gcp-get-secret
The simple utility can be used the configure the environment of an application with values from the Google Secret Manager

## How does it work?
It is simple. Specify one or more environment variables with a URI of the gcp: protocol, as follows:

```
export MYSQL_PASSWORD=gcp:///mysql_root_password'
gcp-get-secret bash -c 'echo $MYSQL_PASSWORD'
```
the utility will lookup the value of `mysql_root_password` in the secret manager of the current project and replace 
the environment variable. The program on the command line will be exec'ed with MYSQL\_PASSWORD set to the actual value.

## secret names
The required secret can be specified in the following formats:
- `<name>`
- `<name>/<version>`
- `<project>/<name>`
- `<project>/<name>/<version>`
- `projects/<project>/secrets/<name>/versions/<version>`

## Query parameters
The utility supports the following query parameters:

- default - value if the value could not be retrieved from the parameter store.
- destination - the filename to write the value to. value replaced with file: url.
- chmod - file permissions of the destination, left to default if not specified. recommended 0600.
- template - the template to use for writing the value, defaults to '{{.}}'

If no default nor destination is specified and the parameter is not found, the utility will return an error.
If a default is specified and the parameter is not found, the utility will use the default.
If a destination file exists and no default is specified, the file will be read as the default value.

For example:
```
$ export ORACLE_PASSWORD=gcp://oracle_scott_password?default=tiger&destination=/tmp/password
$ gcp-get-secret bash -c 'echo $ORACLE_PASSWORD'
/tmp/password
$ cat /tmp/password
tiger
```

## template formatting
To format the secret, you can use the `template` query parameter. For example:
```
$ export PGPASSFILE=gcp://postgres_kong_password?template='localhost:5432:kong:kong:{{.}}%0A&destination=$HOME/.pgpass'
$ gcp-get-secret bash -c 'cat $PGPASSFILE'
localhost:5432:kong:kong:@CypJqmqZ@TYQ2GDnUD@MQGuKyhrl!
```

## Environment substitution
The URI may contain an environment variable reference. For example:
```
$ export ENV=dev
$ export 'PASSWORD=gcp:///${ENV}_mysql_root_password'
gcp-get-secret bash -c 'echo $PASSWORD'
```
will print out the value of `dev_mysql_root_password`.

## Dockerfile usage
To idiomatic way to use the utility is as follows:
```
FROM binxio/gcp-get-secret

FROM alpine:3.6
COPY --from=0 /gcp-get-secret /usr/local/bin/

ENV PGPASSWORD=gcp:///postgres_root_password
ENTRYPOINT [ "/usr/local/bin/gcp-get-secret"]
CMD [ "/bin/bash", "-c", "echo $PGPASSWORD"]
```

## installation
If you have golang installed, type:

```
go get github.com/binxio/gcp-get-secret
```

## installation in Docker
With Docker you can use the multi-stage build:

```
FROM binxio/gcp-get-secret

FROM alpine:3.6
COPY --from=0 /gcp-get-secret /usr/local/bin/
```
