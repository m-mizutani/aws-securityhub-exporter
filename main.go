package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/securityhub"
	"github.com/sirupsen/logrus"
)

var Logger = logrus.New()

const (
	timeFormat = "2006-01-02T15:04:05.000Z"
)

type Arguments struct {
	S3Bucket string
	S3Prefix string
	Region   string
	Minutes  time.Duration
}

func putWorker(finding *securityhub.AwsSecurityFinding, args Arguments, svc *s3.S3) (put bool, err error) {
	ts, err := time.Parse(timeFormat, *finding.CreatedAt)
	if err != nil {
		Logger.WithError(err).WithField("finding", *finding).Error("Fail to parse CreatedAt field")
		return
	}

	keyID := strings.Replace(*finding.Id, ":", "_", -1)
	keyID = strings.Replace(keyID, "/", "_", -1)

	s3Key := strings.Join([]string{
		args.S3Prefix, ts.Format("2006/01/02/15/"), keyID, ".json.gz"}, "")

	_, err = svc.HeadObject(&s3.HeadObjectInput{
		Bucket: &args.S3Bucket,
		Key:    &s3Key,
	})

	exists := true
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey:
				exists = false
			case "NotFound":
				exists = false
			default:
				Logger.WithError(err).Fatalf("HeadObject error: %s", aerr.Error())
				return
			}
		} else {
			Logger.WithError(err).Fatalf("HeadObject error")
			return
		}
	}

	if !exists {
		var raw []byte
		raw, err = json.Marshal(*finding)
		if err != nil {
			Logger.WithError(err).WithField("fiding", *finding).Error("Fail to marshal finding")
			return put, err
		}

		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		zw.Write(raw)
		zw.Close()

		_, err = svc.PutObject(&s3.PutObjectInput{
			Body:   bytes.NewReader(buf.Bytes()),
			Bucket: &args.S3Bucket,
			Key:    &s3Key,
		})

		if err != nil {
			Logger.WithError(err).Fatalf("Fail to put log object: %s", s3Key)
			return
		}

		put = true
	}

	return
}

func getFindings(args Arguments) chan *securityhub.AwsSecurityFinding {
	ch := make(chan *securityhub.AwsSecurityFinding, 32)

	ssn := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(args.Region),
	}))
	svc := securityhub.New(ssn)

	end := time.Now().UTC()
	start := end.Add(-(time.Minute * args.Minutes))

	func() {
		defer close(ch)
		var token *string
		for {
			input := securityhub.GetFindingsInput{
				Filters: &securityhub.AwsSecurityFindingFilters{
					CreatedAt: []*securityhub.DateFilter{
						{
							Start: aws.String(start.Format(timeFormat)),
							End:   aws.String(end.Format(timeFormat)),
						},
					},
				},
				NextToken: token,
			}
			Logger.WithField("input", input).Info("Sending query")

			resp, err := svc.GetFindings(&input)
			if err != nil {
				Logger.WithError(err).WithField("resp", resp).Error("Fail to get findings")
				return
			}

			for _, finding := range resp.Findings {
				ch <- finding
			}
			token = resp.NextToken

			if token == nil {
				break
			}
		}
	}()

	return ch
}

func exportFindings(args Arguments) error {
	ssn := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(args.Region),
	}))
	s3svc := s3.New(ssn)

	ch := getFindings(args)

	for finding := range ch {
		if _, err := putWorker(finding, args, s3svc); err != nil {
			Logger.WithError(err).WithField("finding", finding).Info("Recv finding")
		}
	}
	return nil
}

func main() {
	Logger.SetFormatter(&logrus.JSONFormatter{})
	Logger.SetLevel(logrus.InfoLevel)

	strMinutes := os.Getenv("DURATION")
	if strMinutes == "" {
		strMinutes = "10"
	}
	intMinutes, err := strconv.Atoi(strMinutes)
	if err != nil {
		intMinutes = 10
		Logger.WithError(err).WithField("DURATION", strMinutes).Warn("Fail to parse DURATION as integer and set default value")
	}

	lambda.Start(func() error {
		args := Arguments{
			S3Bucket: os.Getenv("S3_BUCKET"),
			S3Prefix: os.Getenv("S3_PREFIX"),
			Region:   os.Getenv("AWS_REGION"),
			Minutes:  time.Duration(intMinutes),
		}
		return exportFindings(args)
	})
}
