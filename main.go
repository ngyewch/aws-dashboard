package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

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
		fmt.Println("- ", *region.RegionName)
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
			fmt.Println("- ", *instance.InstanceId)
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
		fmt.Println("- ", *securityGroup.GroupId, *securityGroup.GroupName)
		securityGroupMap[*securityGroup.GroupId] = *securityGroup
	}

	s3client := s3.New(&aws.Config{Region: aws.String("ap-southeast-1")})

	const billingDataDir = "data/aws-dashboard/billing"
	err = os.MkdirAll(billingDataDir, os.ModeDir)
	if err != nil {
		panic(err)
	}

  listObjectsInput := s3.ListObjectsInput{Bucket: &config.Billing.BucketName}

	fmt.Println("S3 Objects:")

	const suffix string = ".csv.zip"
	const filename string = "-aws-billing-detailed-line-items-with-resources-and-tags-"

	for {
		listObjectsOutput, err := s3client.ListObjects(&listObjectsInput)
		if err != nil {
			panic(err)
		}

		for _, s3Object := range listObjectsOutput.Contents {
			if strings.HasSuffix(*s3Object.Key, suffix) {
				fmt.Println("- ", *s3Object.Key)
				s := strings.TrimSuffix(*s3Object.Key, suffix)
				p := strings.Index(s, filename)
				if p >= 0 {
					accountId := s[:p]
					year_month := s[p + len(filename):]
					parts := strings.Split(year_month, "-")
					year := parts[0]
					month := parts[1]
					fmt.Println(accountId, year, month)

					path := billingDataDir + "/" + *s3Object.Key

					getObjectOutput, err := s3client.GetObject(&s3.GetObjectInput{Bucket: &config.Billing.BucketName, Key: s3Object.Key})
					if err != nil {
						panic(err)
					}

					writer, err := os.Create(path)
					if err != nil {
						panic(err)
					}

					defer writer.Close()

					io.Copy(writer, getObjectOutput.Body)

					writer.Close()

					err = os.Chtimes(path, *s3Object.LastModified, *s3Object.LastModified)
					if err != nil {
						panic(err)
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
