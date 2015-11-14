package main

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type Pair struct {
	Key string
	Value float64
	Percentage float64
}

type PairList []Pair

func (p Pair) String() string {
	return fmt.Sprintf("%s: USD%.2f (%.1f%%)", p.Key, p.Value, p.Percentage)
}

func (p PairList) Len() int {
	return len(p)
}

func (p PairList) Less(i, j int) bool {
	return p[i].Value < p[j].Value
}

func (p PairList) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func processBilling(config Config, attachment *Attachment) {
	var now time.Time = time.Now()

	s3client := s3.New(session.New(), &aws.Config{Region: aws.String(config.General.DefaultRegion)})

	const billingDataDir = "data/aws-dashboard/billing"
	err := os.MkdirAll(billingDataDir, os.ModeDir)
	if err != nil {
		panic(err)
	}

	listObjectsInput := s3.ListObjectsInput{Bucket: &config.Billing.BucketName}

	fmt.Println("Billing:")

	const suffix string = ".csv.zip"
	const filename string = "-aws-billing-detailed-line-items-with-resources-and-tags-"

	shortProductNames := make(map[string]string)
	shortProductNames["AWS Key Management Service"] = "KMS"
	shortProductNames["Amazon DynamoDB"] = "DynamoDB"
	shortProductNames["Amazon Elastic Compute Cloud"] = "EC2"
	shortProductNames["Amazon RDS Service"] = "RDS"
	shortProductNames["Amazon Route 53"] = "Route53"
	shortProductNames["Amazon Simple Notification Service"] = "SNS"
	shortProductNames["Amazon Simple Queue Service"] = "SQS"
	shortProductNames["Amazon Simple Storage Service"] = "S3"

	for {
		listObjectsOutput, err := s3client.ListObjects(&listObjectsInput)
		if err != nil {
			panic(err)
		}

		for _, s3Object := range listObjectsOutput.Contents {
			if strings.HasSuffix(*s3Object.Key, suffix) {
				s := strings.TrimSuffix(*s3Object.Key, suffix)
				p := strings.Index(s, filename)
				if p >= 0 {
					/*
						accountId := strconv.ParseInt(s[:p], 10, 64)
						if err != nil {
							panic(err)
						}
					*/
					year_month := s[p+len(filename):]
					parts := strings.Split(year_month, "-")
					year, err := strconv.ParseInt(parts[0], 10, 32)
					if err != nil {
						panic(err)
					}
					month, err := strconv.ParseInt(parts[1], 10, 32)
					if err != nil {
						panic(err)
					}

					path := billingDataDir + "/" + *s3Object.Key

					fileInfo, err := os.Stat(path)
					if ((err != nil) && os.IsNotExist(err)) || ((err == nil) && !fileInfo.ModTime().Equal(*s3Object.LastModified)) {

						fmt.Println("- Downloading", *s3Object.Key)

						getObjectOutput, err := s3client.GetObject(&s3.GetObjectInput{Bucket: &config.Billing.BucketName, Key: s3Object.Key})
						if err != nil {
							panic(err)
						}

						writer, err := os.Create(path)
						if err != nil {
							panic(err)
						}

						defer writer.Close()

						_, err = io.Copy(writer, getObjectOutput.Body)
						if err != nil {
							panic(err)
						}

						err = writer.Close()
						if err != nil {
							panic(err)
						}

						err = os.Chtimes(path, *s3Object.LastModified, *s3Object.LastModified)
						if err != nil {
							panic(err)
						}
					}

					zipReader, err := zip.OpenReader(path)
					if err != nil {
						panic(err)
					}
					defer zipReader.Close()

					var totalCost float64 = 0
					costMap := make(map[string]float64)
					var lastRecordTime time.Time
					for _, f := range zipReader.File {
						fileReader, err := f.Open()
						if err != nil {
							panic(err)
						}
						defer fileReader.Close()

						csvReader := csv.NewReader(fileReader)

						headerMap := make(map[string]int)
						record, err := csvReader.Read()
						if err != nil {
							if err != io.EOF {
								panic(err)
							}
						} else {
							for index, column := range record {
								headerMap[column] = index
							}
						}

						for {
							record, err := csvReader.Read()
							if err == io.EOF {
								break
							}
							if err != nil {
								panic(err)
							}

							productName := record[headerMap["ProductName"]]
							if productName != "" {
								//productName := record[headerMap["ProductName"]]
								usageStartDate, err := time.Parse("2006-01-02 15:04:05", record[headerMap["UsageStartDate"]])
								if err != nil {
									panic(err)
								}
								cost, err := strconv.ParseFloat(record[headerMap["Cost"]], 64)
								if err != nil {
									panic(err)
								}

								totalCost += cost
								costMap[productName] += cost
								lastRecordTime = usageStartDate

								//fmt.Println(usageStartDate, productName, cost)
							}
						}
					}

					if (now.Year() == int(year)) && (now.Month() == time.Month(month)) {
						startOfCurrentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
						startOfNextMonth := startOfCurrentMonth.AddDate(0, 1, 0)
						monthDuration := startOfNextMonth.Sub(startOfCurrentMonth)
						currentDuration := lastRecordTime.Sub(startOfCurrentMonth)
						estimatedCost := totalCost * monthDuration.Hours() / (currentDuration.Hours() + 1)
						fmt.Printf("- %04d-%02d USD%.2f to date, estimated cost USD%.2f\n", year, month, totalCost, estimatedCost)
						attachment.Fields = append(attachment.Fields, Field{Title: fmt.Sprintf("%04d-%02d", year, month), Value: fmt.Sprintf("USD%.2f", totalCost), Short: true})
						attachment.Fields = append(attachment.Fields, Field{Title: fmt.Sprintf("%04d-%02d (est.)", year, month), Value: fmt.Sprintf("USD%.2f", estimatedCost), Short: true})

						var pl PairList
						for key, value := range costMap {
							if value < 0.01 {
								continue
							}
							shortProductName := shortProductNames[key]
							if shortProductName == "" {
								shortProductName = key
							}
							pl = append(pl, Pair{shortProductName, value, (value / totalCost) * 100})
						}
						sort.Sort(sort.Reverse(pl))
						var arr []string
						for _, p := range pl {
							arr = append(arr, p.String())
						}
						attachment.Fields = append(attachment.Fields, Field{Title: fmt.Sprintf("%04d-%02d (detailed)", year, month), Value: strings.Join(arr, "\n"), Short: false})
					} else {
						fmt.Printf("- %04d-%02d USD%.2f\n", year, month, totalCost)
						attachment.Fields = append(attachment.Fields, Field{Title: fmt.Sprintf("%04d-%02d", year, month), Value: fmt.Sprintf("USD%.2f", totalCost), Short: true})
					}
				}
			}
		}

		if !*listObjectsOutput.IsTruncated {
			break
		}

		listObjectsInput.Marker = listObjectsOutput.NextMarker
	}
}
