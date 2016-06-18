# logshipper

Logshipper is a simple utility to wrap another application, intercept its
`stdout` and `stderr` output, and ship that to a JSON-encoded-log logging server
(such as `logstash`).

Logshipper itself simply wraps Alan Shreve's 
[log15](https://github.com/inconshreveable/log15) tool.

The primary use case for `logshipper` is to wrap legacy applications which can
dump their logs to `stdout` or `stderr` for use in a container-based deployment
with centralised logging.

`logshipper` accept the following arguments:

  - `-application` (optional) set the `application` tag with the provided name
  - `-binary` (required) set the program to execute
  - `-arguments` (optional) set any additional argument to pass to the main
    executable
  - `-prefix` (optional) set some text which will be prepended to any log
    entries
  - `-loghost` (required) set the loghost and port to which the logs should be
    sent (ex: `log.mydomain.com:9514`)

