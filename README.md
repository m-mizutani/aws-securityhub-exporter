# aws-securityhub-exporter

`aws-securityhub-exporter` exports [findings](https://docs.aws.amazon.com/securityhub/latest/userguide/securityhub-findings.html) of AWS Security Hub to S3 bucket as objects. In this moment (2019.7), AWS does not provide feature to export or push created findings to other AWS services and security products. Then Security Hub can not be integrated with security alert management service and product, such as SIEM (Security Information & Event Manager). `aws-securityhub-exporter` provides source code and template of AWS CloudFormation to build Lambda function to export findings to S3 bucket. The Lambda function is invoked periodically every 1 minute and fetches fidings in last 10 minutes (default). After that, Lambda put them to S3 bucket idempotently. Then You can integrate your security service/product with S3 event notification feature.

## Prerequisite

- go >= 1.12
- aws-cli >= 1.16.190
- GNU make >= 3.81
- jq >= 1.6
- jsonnet >= 0.13.0

## Deploy

### Clone repository

```bash
$ git clone https://github.com/m-mizutani/aws-securityhub-exporter.git
```

NOTE: You do not need to change directory into the repository.

### Create config files

2 jsonnet files, `deploy.jsonnet` and `stack.jsonnet` are required.

`deploy.jsonnet` : Configurations for deploy CloudFormation.
```jsonnet
{
    StackName: 'your-stack-name',
    // This bucket will be put materials (zipped code) of the stack
    CodeS3Bucket: 'your-bucket-to-save-code',
    // Materials are put under the prefix
    CodeS3Prefix: "materials",
    Region: 'ap-northeast-1',
}
```

`stack.jsonnet` : Configurations for CloudFomation stack.
```jsonnet
local template = import 'template.libsonnet';

template.build(
    // Required
    S3Bucket='your-s3-bucket',
    // Optional
    S3Prefix='your-prefix/',
    // Optional, IAM Role for Lambda is created automatically if you do not set the item
    LambdaRoleArn='arn:aws:iam::1234567890:role/YourLambdaRole'
)
```

### Run deploy procedure

```bash
$ ls
aws-securityhub-exporter
deploy.jsonnet
stack.jsonnet
$ make deploy -f aws-securityhub-exporter/Makefile
```

## License

- [The 3-Clause BSD License](https://opensource.org/licenses/BSD-3-Clause)
- Author: Masayoshi Mizutani <mizutani@sfc.wide.ad.jp>
