% horus-query(1)

NAME
====

**horus-query** - Runs a single snmp polling job and prints the result to stdout.

SYNOPSIS
========

| **horus-query** \[**-c**|**-h**|**-p**|**-v**] \[**-d** _level_] \[**--dsn** _url_] \[**-i** _value_] \[**-r** _json_] \[**-s** _id,..._] \[**-t** _id,..._]

DESCRIPTION
===========

**horus-query** is a test tool that builds an snmp polling request for a device from db, runs it directly, and displays to stdout the json result that would be sent to kafka.

Options
-------

-c, --compact

:   Prints the result data in compact json format, without indentation.

-d, --debug

:   Specifies the debug level from 1 to 3. Defaults to 0 (disabled).

    --dsn

:   Specifies the postgres db connection DSN like `postgres://horus:secret@localhost/horus`.

-h, --help

:   Prints a help message.

-i, --id

:   Specifies the database id of the device to poll.

-p, --print-query

:   Prints the json request (as it would be sent to the agent) before executing it.

-r, --request

:   Reads the json request to execute from a file intead of querying the db.

-s, --scalar

:   Specifies a CSV list of ids of scalar measures to query. If empty all measures defined in db are polled (default). Set to 0 to poll no scalar measure.

-t, --indexed

:   Specifies a CSV list of ids of indexed measures to query. If empty all measures db are polled (default). Set to 0 to poll no indexed measure.

-v, --version

:   Prints the current version and build date.

BUGS
====

See GitHub Issues: <https://github.com/kosctelecom/horus/issues>

AUTHOR
======

Vallimamod Abdullah <vma@sip.solutions>

SEE ALSO
========

**horus-dispatcher(1)**, **horus-agent(1)**
