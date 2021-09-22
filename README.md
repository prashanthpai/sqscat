# sqscat

sqscat polls AWS SQS queue for messages and streams them to stdout with
newline as the delimiter. The output can be piped further to other Unix tools.

Inspired by [kafkacat](https://github.com/edenhill/kafkacat).

### Install

You can download built binaries for Mac, Linux and Windows from the
[releases](https://github.com/ppai-plivo/sqscat/releases) page.

### Usage

**Note:** sqscat uses AWS SDK which will automatically load access and region
configuration from environment variables, AWS shared configuration file
(`~/.aws/config`), and AWS shared credentials file (`~/.aws/credentials`).

```sh
$ sqscat --help
Usage:
  sqscat [OPTIONS] queue-name

Application Options:
  -v, --version       Print version and exit
  -c, --concurrency=  Number of concurrent SQS pollers; Defaults to 10 x Num. of CPUs
  -d, --delete        Delete received messages
  -n, --num-messages= Receive specified number of messages and exit; This limits concurrency to 1
  -t, --timeout=      Exit after specified number of seconds

Help Options:
  -h, --help          Show this help message
```

#### Examples

Keep reading json payloads from queue and extract individual fields using [jq](https://stedolan.github.io/jq/):

```sh
$ sqscat my-sqs-queue | jq --unbuffered '.employee.email'
```

Read 25 messages from queue and exit:

```sh
$ sqscat -n 25 my-sqs-queue
```

Keep reading from queue for 10 seconds and exit:

```sh
$ sqscat -t 10 my-sqs-queue
```

Delete received messages after they have been printed to stdout:

```sh
$ sqscat -d my-sqs-queue
```

### TODO

1. Add producer mode i.e read from stdin and send to SQS.
