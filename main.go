package main

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	//"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/ghodss/yaml"
)

type Billing struct {
	BucketName string `json:bucketName`
}

type Config struct {
	Billing Billing `json:billing`
}

func main() {
	configData, err := ioutil.ReadFile("aws-dashboard.yaml")
	if err != nil {
		panic(err)
	}

	var config Config
	err = yaml.Unmarshal(configData, &config)
	if err != nil {
		panic(err)
	}

	ec2client := ec2.New(&aws.Config{Region: aws.String("ap-southeast-1")})
	regions, err := ec2client.DescribeRegions(&ec2.DescribeRegionsInput{})
	if err != nil {
		panic(err)
	}
	fmt.Println("Regions:")
	for _, region := range regions.Regions {
		fmt.Println("-", *region.RegionName)
	}

	ec2client = ec2.New(&aws.Config{Region: aws.String("ap-southeast-1")})

	instances, err := ec2client.DescribeInstances(nil)
	if err != nil {
		panic(err)
	}

	fmt.Println("Instances:")
	instanceMap := make(map[string]ec2.Instance)
	for _, reservation := range instances.Reservations {
		for _, instance := range reservation.Instances {
			fmt.Println("-", *instance.InstanceId)
			instanceMap[*instance.InstanceId] = *instance
		}
	}

	securityGroups, err := ec2client.DescribeSecurityGroups(nil)
	if err != nil {
		panic(err)
	}

	fmt.Println("Security Groups:")
	securityGroupMap := make(map[string]ec2.SecurityGroup)
	for _, securityGroup := range securityGroups.SecurityGroups {
		fmt.Println("-", *securityGroup.GroupId, *securityGroup.GroupName)
		securityGroupMap[*securityGroup.GroupId] = *securityGroup
	}

	s3client := s3.New(&aws.Config{Region: aws.String("ap-southeast-1")})

	const billingDataDir = "data/aws-dashboard/billing"
	err = os.MkdirAll(billingDataDir, os.ModeDir)
	if err != nil {
		panic(err)
	}

	listObjectsInput := s3.ListObjectsInput{Bucket: &config.Billing.BucketName}

	fmt.Println("Billing:")

	const suffix string = ".csv.zip"
	const filename string = "-aws-billing-detailed-line-items-with-resources-and-tags-"

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
					//accountId := s[:p]
					year_month := s[p+len(filename):]
					parts := strings.Split(year_month, "-")
					year := parts[0]
					month := parts[1]

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
								/*
								productName := record[headerMap["ProductName"]]
								usageStartDate, err := time.Parse("2006-01-02 15:04:05", record[headerMap["UsageStartDate"]])
								if err != nil {
									panic(err)
								}
								*/
								cost, err := strconv.ParseFloat(record[headerMap["Cost"]], 64)
								if err != nil {
									panic(err)
								}

								totalCost += cost
								//fmt.Println(usageStartDate, productName, cost)
							}
						}
					}

					fmt.Printf("- %s-%s USD%.2f\n", year, month, totalCost)
				}
			}
		}

		if !*listObjectsOutput.IsTruncated {
			break
		}

		listObjectsInput.Marker = listObjectsOutput.NextMarker
	}
}
