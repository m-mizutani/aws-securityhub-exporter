{
  build(S3Bucket, S3Prefix='', LambdaRoleArn='', Duration='10'):: {
    AWSTemplateFormatVersion: '2010-09-09',
    Transform: 'AWS::Serverless-2016-10-31',

    Resources: {
      // --------------------------------------------------------
      // Lambda functions
      Handler: {
        Type: 'AWS::Serverless::Function',
        Properties: {
          CodeUri: 'build',
          Handler: 'main',
          Runtime: 'go1.x',
          Timeout: 60,
          MemorySize: 128,
          Role: (if LambdaRoleArn != '' then LambdaRoleArn else { Ref: 'LambdaRole' }),
          Environment: {
            Variables: {
              S3_BUCKET: S3Bucket,
              S3_PREFIX: S3Prefix,
              DURATION: Duration,
            },
          },

          Events: {
            Every1min: {
              Type: 'Schedule',
              Properties: {
                Schedule: 'rate(1 minute)',
              },
            },
          },
        },
      },
    } + (if LambdaRoleArn != '' then {} else {
           // --------------------------------------------------------
           // Lambda IAM role
           LambdaRole: {
             Type: 'AWS::IAM::Role',
             Condition: 'LambdaRoleRequired',
             Properties: {
               AssumeRolePolicyDocument: {
                 Version: '2012-10-17',
                 Statement: [
                   {
                     Effect: 'Allow',
                     Principal: { Service: ['lambda.amazonaws.com'] },
                     Action: ['sts:AssumeRole'],
                   },
                 ],
                 Path: '/',
                 ManagedPolicyArns: ['arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole'],
                 Policies: [
                   {
                     PolicyName: 'AccessSecurityHub',
                     PolicyDocument: {
                       Version: '2012-10-17',
                       Statement: [
                         {
                           Effect: 'Allow',
                           Action: ['s3:ListBucket', 's3:PutObject', 's3:GetObject'],
                           Resource: [
                             'arn:aws:s3:::' + S3Bucket,
                             'arn:aws:s3:::' + S3Bucket + '/' + S3Prefix + '*',
                           ],
                         },
                       ],
                     },
                   },
                   {
                     PolicyName: 'AccessSecurityHub',
                     PolicyDocument: {
                       Version: '2012-10-17',
                       Statement: [
                         {
                           Effect: 'Allow',
                           Action: ['securityhub:GetFindings'],
                           Resource: ['*'],
                         },
                       ],
                     },
                   },
                 ],
               },
             },
           },
         }),
  },
}
