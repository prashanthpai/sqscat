# sqscat

sqscat is "netcat for SQS". You can use sqscat to receive from and send
messages to SQS queue. sqscat uses newline as a delimiter between messages.

sqscat automatically detects and selects its mode (receive/send) depending
on the terminal or pipe type:

* If data is being piped into sqscat (stdin), it will send messages to SQS.
* If data is being piped from sqscat (stdout) it will receive messages from SQS.

sqscat is inspired by [kafkacat](https://github.com/edenhill/kafkacat).

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

**Receive mode:**

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

**Send mode:**

Parse json and keep sending messages to the queue. sqscat will exit after the
input pipe has been closed.

```sh
$ cat employees.json | jq --unbuffered '.employee.email' | ./sqscat dev-ppai-temp
```

Send 25 messages to the queue and exit:

```sh
$ cat employees.json | jq --unbuffered '.employee.email' | sqscat -n 25 my-sqs-queue
```
