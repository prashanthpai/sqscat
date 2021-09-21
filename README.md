# sqscat

Poll AWS SQS payloads for inspection.

## Usage

```sh
$ ./sqscat --help
Usage:
  sqscat [OPTIONS] queue-name

Application Options:
  -c, --concurrency= Number of concurrent SQS pollers; Defaults to 10 x Num. of CPUs

Help Options:
  -h, --help         Show this help message
```
