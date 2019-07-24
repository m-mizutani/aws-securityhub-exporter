package main_test

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/securityhub"
	"github.com/google/uuid"
	main "github.com/m-mizutani/aws-securityhub-exporter"
	"github.com/stretchr/testify/assert"
)

type testConfig struct {
	S3Bucket string
	S3Prefix string
	Region   string
	Minutes  int
}

var testCfg testConfig

func loadTestConfig() testConfig {
	var cfg testConfig
	raw, err := ioutil.ReadFile("test.json")
	if err != nil {
		log.Fatal("Fail to read test.json", err)
	}

	if err := json.Unmarshal(raw, &cfg); err != nil {
		log.Fatal("Fail to unmarshal test.json")
	}

	return cfg
}

func init() {
	testCfg = loadTestConfig()
}

func TestGetFindings(t *testing.T) {
	args := main.Arguments{
		Region:  testCfg.Region,
		Minutes: time.Duration(testCfg.Minutes),
	}
	ch := main.GetFindings(args)

	var findings []*securityhub.AwsSecurityFinding
	for finding := range ch {
		findings = append(findings, finding)
	}

	assert.NotEqual(t, 0, len(findings))
}

func TestPutWorker(t *testing.T) {
	finding := securityhub.AwsSecurityFinding{
		Id:        aws.String(uuid.New().String()),
		CreatedAt: aws.String("2019-07-20T13:22:13.933Z"),
	}
	args := main.Arguments{
		Region:   testCfg.Region,
		S3Bucket: testCfg.S3Bucket,
		S3Prefix: testCfg.S3Prefix,
	}
	ssn := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(args.Region),
	}))
	s3svc := s3.New(ssn)

	put, err := main.PutWorker(&finding, args, s3svc)
	assert.NoError(t, err)
	assert.True(t, put)

	put, err = main.PutWorker(&finding, args, s3svc)
	assert.NoError(t, err)
	assert.False(t, put) // Should not be put twice by idempotence
}
