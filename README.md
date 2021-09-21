# sqscat

sqscat polls AWS SQS queue for messages and streams them to stdout.
It does not delete the received messages.

### Install

You can download built binaries for Mac, Linux and Windows from the
[releases](https://github.com/ppai-plivo/sqscat/releases) page.

### Usage

**Note:** sqscat uses AWS SDK which will automatically load access and region
configuration from environment variables, AWS shared configuration file
(`~/.aws/config`), and AWS shared credentials file (`~/.aws/credentials`).

```sh
$ ./sqscat --help
Usage:
  sqscat [OPTIONS] queue-name

Application Options:
  -c, --concurrency= Number of concurrent SQS pollers; Defaults to 10 x Num. of CPUs

Help Options:
  -h, --help         Show this help message
```

sqscat streams messages from SQS queue to STDOUT with newline as the delimiter.
This can be piped further to other unix tools. For example:

```sh
$ sqscat my-sqs-queue | jq --unbuffered '.employee.email'
```
