package main

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

func processBilling(config Config) {
	s3client := s3.New(&aws.Config{Region: aws.String(config.General.DefaultRegion)})

	const billingDataDir = "data/aws-dashboard/billing"
	err := os.MkdirAll(billingDataDir, os.ModeDir)
	if err != nil {
		panic(err)
	}

	listObjectsInput := s3.ListObjectsInput{Bucket: &config.Billing.BucketName}

	fmt.Println("Billing:")

	const suffix string = ".csv.zip"
	const filename string = "-aws-billing-detailed-line-items-with-resources-and-tags-"

	var now time.Time = time.Now()

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
					if (err == nil) && !fileInfo.ModTime().Equal(*s3Object.LastModified) {

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

							if record[headerMap["ProductName"]] != "" {
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
					} else {
						fmt.Printf("- %04d-%02d USD%.2f\n", year, month, totalCost)
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
