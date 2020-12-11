<a id="markdown-runsqs---prepackaged-runtime-helper-for-aws-sqs" name="runsqs---prepackaged-runtime-helper-for-aws-sqs"></a>
# runsqs - Prepackaged Runtime Helper For AWS SQS
[![GoDoc](https://godoc.org/github.com/asecurityteam/runsqs?status.svg)](https://godoc.org/github.com/asecurityteam/runsqs)
[![Build Status](https://travis-ci.com/asecurityteam/runsqs.png?branch=master)](https://travis-ci.com/asecurityteam/runsqs)
[![codecov.io](https://codecov.io/github/asecurityteam/runsqs/coverage.svg?branch=master)](https://codecov.io/github/asecurityteam/runsqs?branch=master)
<!-- TOC -->

- [runsqs - Prepackaged Runtime Helper For AWS SQS](#runsqs---prepackaged-runtime-helper-for-aws-sqs)
    - [Overview](#overview)
    - [Quick Start](#quick-start)
    <!-- - [Details](#details)
        - [Configuration](#configuration)
            - [YAML](#yaml)
            - [ENV](#env)
        - [Logging](#logging)
        - [Metrics](#metrics) -->
    - [Status](#status)
    - [Contributing](#contributing)
        - [Building And Testing](#building-and-testing)
        - [License](#license)
        - [Contributing Agreement](#contributing-agreement) -->

<!-- TOC -->

<a id="markdown-overview" name="overview"></a>
## Overview

This project is a tool bundle for running a service that interacts with AWS SQS written in go. It comes with
an opinionated choice of logger, metrics client, and configuration parsing. The benefits
are a simple abstraction around the aws-sdk api for consuming from an SQS or producing to an SQS.

<a id="markdown-quick-start" name="quick-start"></a>
## Quick Start

```golang
package main

import (
    "net/http"
    "fmt"

    "github.com/asecurityteam/runsqs"
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/sqs"
)

type BasicConsumer string

func (m BasicConsumer) ConsumeMessage(ctx context.Context, message []byte) error {
    fmt.Println(string(message))
    fmt.Println(m)
    return nil
}

func main() {
    // create a new aws session, and establish a SQS instance to connect to.
    // aws new sessions by default reads AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY as environment variables to use
	var sesh = session.Must(session.NewSession())
	queue := sqs.New(sesh, &aws.Config{
		Region:     aws.String("us-west-2"),
		HTTPClient: http.DefaultClient,
		Endpoint:   aws.String("www.aws.com"),
    })



    consumer := runsqs.DefaultSQSQueueConsumer{
        Queue: queue,
        QueueURL: "www.aws.com/url/to/queue",
        MessageConsumer: BasicConsumer{"consooooom"},
        LogFn: runsqs.LoggerFromContext,
    }

    producer := runsqs.DefaultSQSProducer{
        Queue: queue,
        QueueURL: "www.aws.com/url/to/queue",
    }

    go producer.ProduceMessage([]byte("incoming sqs message"))

    // Run the SQS consumer.
    if err := consumer.StartConsuming(); err != nil {
		panic(err.Error())
    }

    // expected output:
    // "incoming sqs message"
    // "consooooom"
}
```

<a id="markdown-status" name="status"></a>
## Status

This project is in incubation which the interfaces and implementations are subject to change.

<a id="markdown-contributing" name="contributing"></a>
## Contributing

<a id="markdown-building-and-testing" name="building-and-testing"></a>
### Building And Testing

We publish a docker image called [SDCLI](https://github.com/asecurityteam/sdcli) that
bundles all of our build dependencies. It is used by the included Makefile to help make
building and testing a bit easier. The following actions are available through the Makefile:

-   make dep

    Install the project dependencies into a vendor directory

-   make lint

    Run our static analysis suite

-   make test

    Run unit tests and generate a coverage artifact

-   make integration

    Run integration tests and generate a coverage artifact

-   make coverage

    Report the combined coverage for unit and integration tests

<a id="markdown-license" name="license"></a>
### License

This project is licensed under Apache 2.0. See LICENSE.txt for details.

<a id="markdown-contributing-agreement" name="contributing-agreement"></a>
### Contributing Agreement

Atlassian requires signing a contributor's agreement before we can accept a patch. If
you are an individual you can fill out the [individual
CLA](https://na2.docusign.net/Member/PowerFormSigning.aspx?PowerFormId=3f94fbdc-2fbe-46ac-b14c-5d152700ae5d).
If you are contributing on behalf of your company then please fill out the [corporate
CLA](https://na2.docusign.net/Member/PowerFormSigning.aspx?PowerFormId=e1c17c66-ca4d-4aab-a953-2c231af4a20b).
