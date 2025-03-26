<a id="markdown-runsqs---prepackaged-runtime-helper-for-aws-sqs" name="runsqs---prepackaged-runtime-helper-for-aws-sqs"></a>
# runsqs - Prepackaged Runtime Helper For AWS SQS
[![GoDoc](https://godoc.org/github.com/asecurityteam/runsqs?status.svg)](https://godoc.org/github.com/asecurityteam/runsqs)

[![Bugs](https://sonarcloud.io/api/project_badges/measure?project=asecurityteam_runsqs&metric=bugs)](https://sonarcloud.io/dashboard?id=asecurityteam_runsqs)
[![Code Smells](https://sonarcloud.io/api/project_badges/measure?project=asecurityteam_runsqs&metric=code_smells)](https://sonarcloud.io/dashboard?id=asecurityteam_runsqs)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=asecurityteam_runsqs&metric=coverage)](https://sonarcloud.io/dashboard?id=asecurityteam_runsqs)
[![Duplicated Lines (%)](https://sonarcloud.io/api/project_badges/measure?project=asecurityteam_runsqs&metric=duplicated_lines_density)](https://sonarcloud.io/dashboard?id=asecurityteam_runsqs)
[![Lines of Code](https://sonarcloud.io/api/project_badges/measure?project=asecurityteam_runsqs&metric=ncloc)](https://sonarcloud.io/dashboard?id=asecurityteam_runsqs)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=asecurityteam_runsqs&metric=sqale_rating)](https://sonarcloud.io/dashboard?id=asecurityteam_runsqs)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=asecurityteam_runsqs&metric=alert_status)](https://sonarcloud.io/dashboard?id=asecurityteam_runsqs)
[![Reliability Rating](https://sonarcloud.io/api/project_badges/measure?project=asecurityteam_runsqs&metric=reliability_rating)](https://sonarcloud.io/dashboard?id=asecurityteam_runsqs)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=asecurityteam_runsqs&metric=security_rating)](https://sonarcloud.io/dashboard?id=asecurityteam_runsqs)
[![Technical Debt](https://sonarcloud.io/api/project_badges/measure?project=asecurityteam_runsqs&metric=sqale_index)](https://sonarcloud.io/dashboard?id=asecurityteam_runsqs)
[![Vulnerabilities](https://sonarcloud.io/api/project_badges/measure?project=asecurityteam_runsqs&metric=vulnerabilities)](https://sonarcloud.io/dashboard?id=asecurityteam_runsqs)


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
	"context"
	"fmt"
	"time"

	"github.com/asecurityteam/runsqs/v4"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type BasicConsumer struct{}

func (m BasicConsumer) ConsumeMessage(ctx context.Context, message *types.Message) runsqs.SQSMessageConsumerError {
	fmt.Println(string(*message.Body))
	return nil
}

func (m BasicConsumer) DeadLetter(ctx context.Context, message *types.Message) {
	fmt.Println("deadletter")
}

func main() {
	// create a new SQS config, and establish a SQS instance to connect to.
	// aws new sessions by default reads AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY as environment variables to use
	ctx := context.Background()

	consumerComponent := runsqs.DefaultSQSQueueConsumerComponent{}
	consumerConfig := consumerComponent.Settings()
	consumerConfig.QueueURL = "www.aws.com/url/to/queue"
	consumerConfig.AWSEndpoint = "www.aws.com"
	consumerConfig.QueueRegion = "us-east-1"
	consumerConfig.PollInterval = 1 * time.Second

	consumer, err := consumerComponent.New(ctx, consumerConfig)
	if err != nil {
		panic(err.Error())
	}

	consumer.MessageConsumer = BasicConsumer{}

	producerComponent := runsqs.DefaultSQSProducerComponent{}
	producerConfig := producerComponent.Settings()
	producerConfig.QueueURL = "www.aws.com/url/to/queue"
	producerConfig.AWSEndpoint = "www.aws.com"
	producerConfig.QueueRegion = "us-east-1"

	producer, err := producerComponent.New(ctx, producerConfig)
	if err != nil {
		panic(err.Error())
	}

	go producer.ProduceMessage(ctx, &sqs.SendMessageInput{
		MessageBody: aws.String("incoming sqs message"),
		QueueUrl:    aws.String(producer.QueueURL()),
	})

	// Run the SQS consumer.
	if err := consumer.StartConsuming(ctx); err != nil {
		panic(err.Error())
	}

	// expected output:
	// "incoming sqs message"
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
